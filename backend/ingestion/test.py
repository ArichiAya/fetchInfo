# coding=utf-8
from __future__ import print_function, absolute_import
from gm.api import *


def init(context):
    subscribe(symbols='SHSE.600519', frequency='tick')


def on_tick(context, tick):

    # 打印tick数据
    print(tick)


if __name__ == '__main__':
    '''
        strategy_id策略ID, 由系统生成
        filename文件名, 请与本文件名保持一致
        mode运行模式, 实时模式:MODE_LIVE回测模式:MODE_BACKTEST
        token绑定计算机的ID, 可在系统设置-密钥管理中生成
        backtest_start_time回测开始时间
        backtest_end_time回测结束时间
        backtest_adjust股票复权方式, 不复权:ADJUST_NONE前复权:ADJUST_PREV后复权:ADJUST_POST
        backtest_initial_cash回测初始资金
        backtest_commission_ratio回测佣金比例
        backtest_slippage_ratio回测滑点比例
    '''
    run(strategy_id='38aab753-9bcf-11f1-a452-00ff0cd61834',
        filename='test.py',
        mode=MODE_BACKTEST,
        token='073faa373d7a42b7925aa1f15c1781a266d09183',
        backtest_start_time='2026-10-08 08:00:00',
        backtest_end_time='2026-10-08 16:00:00',
        backtest_adjust=ADJUST_PREV,
        backtest_initial_cash=10000000,
        backtest_commission_ratio=0.0001,
        backtest_slippage_ratio=0.0001)
