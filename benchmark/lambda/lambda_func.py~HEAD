import json
import time
import os

from tests import *

def run_cmd(cmd):
    return os.popen(cmd).read().strip("\n")


def handler(event):
    # start timer
#    tm_st = time.time() * 1000

    # the map for converting parameters to tests
    CMD_2_FUNC = {
        "sleep": 0,
        "run": run_cmd,
        "io": ioload_test,
        "net": network_test,
        "cpu": cpu_test,
        "mem": mem_test,
#        "django": django_test,
#        "matplotlib_numpy": matplotlib_numpy_test,
#        "pandas_numpy": pandas_numpy_test,
#        "pip_numpy": pip_numpy_test,
#        "setuptools": setuptools_test,
    }

    cmds = event["cmds"]


    basic_info = []
    for cmd in cmds:
        # find the tests to run based on the parameter
        func = CMD_2_FUNC[cmd[0]]
        para = cmd[1:]
        try:
            res = func(*para)
        except BaseException:
            res = None
        # collect all results
        basic_info.append(str(res))

    #tm_ed = time.time() * 1000
    #timing_info = [fstr(tm_st), fstr(tm_ed), fstr(tm_ed - tm_st)]

    res = '#'.join(basic_info)
    return res


#########################

