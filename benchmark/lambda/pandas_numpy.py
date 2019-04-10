#!/usr/bin/python
import numpy as np
import time
import pandas as pd

def pandas_numpy_test():
    s = pd.Series([1, 3, 5, np.nan, 6, 8])
    dates = pd.date_range('20130101', periods=6)
    df = pd.DataFrame(np.random.randn(10, 4))
    pieces = [df[:3], df[3:7], df[7:]]
    pd.concat(pieces)
    time.sleep(0.1)
