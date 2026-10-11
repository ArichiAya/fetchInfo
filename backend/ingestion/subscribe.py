"""
subscribe.py 交易日当天程序;连接掘金,获取数据并发送给NATS
nats-py v2 是纯异步客户端,所有 NATS 操作必须在 asyncio 事件循环中 await
"""
import asyncio
import queue
import threading
import time
import os
import sys

from gm.api import *
import nats
from pbs.message_pb2 import Tick, Bar, Quote

STRATEGY_ID = '38aab753-9bcf-11f1-a452-00ff0cd61834'
TOKEN = '073faa373d7a42b7925aa1f15c1781a266d09183'
CODES = ['SHSE.600519']
NATS_URL = 'nats://192.168.80.131:4222'

QUEUE_MAXSIZE = 10000
STOP = object()  # 队列中放入这个对象表示停止


async def _connect():
    """在事件循环内建立NATS连接并创建stream"""
    nc = await nats.connect(NATS_URL)
    js = nc.jetstream()
    await js.add_stream(name='tick_stream', subjects=['tick.>'])
    await js.add_stream(name='bar_stream', subjects=['bar.>'])
    return nc, js


def init(context):
    """初始化"""
    # 独立线程运行 asyncio 事件循环,NATS 的协程全部提交到这个 loop 执行
    context.loop = asyncio.new_event_loop()
    context.loop_thread = threading.Thread(
        target=context.loop.run_forever,
        daemon=False)  #daemon为False确保线程在主线程退出前完成任务
    context.loop_thread.start()

    # 在 loop 内建立连接,阻塞当前线程直到连接就绪(init 只在启动时执行一次)
    context.nc, context.js = asyncio.run_coroutine_threadsafe(
        _connect(), context.loop).result(timeout=10)

    # 发送队列 + 后台发送线程:
    # on_tick/on_bar 运行在掘金SDK内部的行情线程上,阻塞会卡住行情接收,
    # 因此回调里只入队,由独立线程把协程提交到 loop 中 await
    context.msg_queue = queue.Queue(maxsize=QUEUE_MAXSIZE)
    context.pub_thread = threading.Thread(
        target=_publisher, args=(context, ),
        daemon=False)  #daemon为False确保线程在主线程退出前完成任务
    context.pub_thread.start()

    #订阅CODES中所有股票的tick级和bar级数据
    for code in CODES:
        subscribe(symbols=code, frequency='tick', count=1)
        subscribe(symbols=code, frequency='300s', count=1)


def _publisher(context):
    """后台线程:从队列取消息,把 publish 协程提交到 loop 并等待结果"""
    loop = context.loop
    while True:
        item = context.msg_queue.get()

        try:
            if item is STOP:
                print("Publisher thread received STOP signal. Exiting.")
                break

            subject, symbol, payload = item

            while True:
                try:
                    # publish 协程交给 loop 去 await,阻塞只发生在本线程
                    ack = asyncio.run_coroutine_threadsafe(
                        context.js.publish(subject, payload, timeout=1),
                        loop).result(timeout=2)

                    print(
                        f"Published {subject} data for {symbol} to NATS with ack: {ack}"
                    )
                    break  # 成功发送后跳出重试循环

                except Exception as e:
                    print(
                        f"Failed to publish {subject} data for {symbol}: {e}.retrying..."
                    )
                    time.sleep(1)  # 连接断开时快速失败会刷屏,限流后重试
        finally:
            context.msg_queue.task_done()

    print("Publisher thread exited")


count = 0


def on_tick(context, tick):
    """
    处理tick数据
    """
    #将tick数据转为protobuf格式
    tick_pb = Tick()
    tick_pb.symbol = tick.symbol
    tick_pb.open = tick.open
    tick_pb.high = tick.high
    tick_pb.low = tick.low
    tick_pb.price = tick.price
    tick_pb.cum_volume = tick.cum_volume
    tick_pb.cum_amount = tick.cum_amount
    tick_pb.trade_type = tick.trade_type
    tick_pb.last_volume = tick.last_volume
    tick_pb.cum_position = tick.cum_position
    tick_pb.last_amount = tick.last_amount
    tick_pb.created_at = tick.created_at.strftime('%Y-%m-%d %H:%M:%S.%f')
    tick_pb.quotes.extend(Quote(**quote) for quote in tick.quotes)
    tick_pb.iopv = tick.iopv
    # 只入队,不阻塞回调线程;真正发送由 _publisher 线程完成
    try:
        context.msg_queue.put_nowait(
            ('tick.new', tick.symbol, tick_pb.SerializeToString()))
        global count
        count += 1
        print(
            f"Enqueued tick for {tick.symbol}. Total ticks enqueued: {count}")
    except queue.Full:
        print(f"publish queue full, dropping tick for {tick.symbol}")


def on_bar(context, bars):
    """
    处理bar数据
    """
    #将bar数据转为protobuf格式
    for bar in bars:
        bar_pb = Bar()
        bar_pb.symbol = bar.symbol
        bar_pb.frequency = bar.frequency
        bar_pb.open = bar.open
        bar_pb.close = bar.close
        bar_pb.high = bar.high
        bar_pb.low = bar.low
        bar_pb.amount = bar.amount
        bar_pb.volume = bar.volume
        bar_pb.bob = bar.bob.strftime('%Y-%m-%d %H:%M:%S.%f')
        bar_pb.eob = bar.eob.strftime('%Y-%m-%d %H:%M:%S.%f')

        try:
            context.msg_queue.put(
                ('bar.new', bar.symbol, bar_pb.SerializeToString()
                 ))  #使用put而不是put_nowait来避免回测时丢失bar数据,在非回测模式下使用put_nowait防止阻塞
        except queue.Full:
            print(f"publish queue full, dropping bar for {bar.symbol}")


def on_backtest_finished(context, indicator):
    """
    回测结束时的回调函数
    """
    print("backtest finished, indicator:", indicator)

    # STOP 放到 Queue 最后
    context.msg_queue.put(STOP)

    # 等待 Publisher 把 STOP 前面的所有消息发送完
    context.pub_thread.join()

    print("all messages published, draining NATS...")

    # 关闭 NATS
    future = asyncio.run_coroutine_threadsafe(context.nc.drain(), context.loop)

    try:
        future.result(timeout=10)
    except Exception as e:
        print(f"NATS drain failed: {e}")

    # 停止 event loop
    context.loop.call_soon_threadsafe(context.loop.stop)

    # 等待 event loop 线程结束
    context.loop_thread.join()

    # 关闭 loop
    context.loop.close()

    print("backtest finished.nats connection closed, event loop stopped.")
    global count
    print(f"Total messages processed: {count}")


def main():
    """
    主函数,启动掘金SDK
    """
    run(strategy_id=STRATEGY_ID,
        filename='subscribe.py',
        mode=MODE_BACKTEST,
        token=TOKEN,
        backtest_start_time='2026-10-09 09:30:00',
        backtest_end_time='2026-10-09 15:00:00',
        backtest_adjust=ADJUST_PREV,
        backtest_initial_cash=10000000,
        backtest_commission_ratio=0.0001,
        backtest_slippage_ratio=0.0001)


if __name__ == '__main__':
    main()
    # import os

    # print("__file__ =", __file__)
    # print("cwd =", os.getcwd())
    # print("absolute =", os.path.abspath(__file__))
    # print(sys.path)
