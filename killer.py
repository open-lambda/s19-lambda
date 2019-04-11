import os, signal
#import memstats

def get_RSS(pid):
   # vss, rss, pss, uss = memstats.measure(pid)
    return "hello"

def kill_process(pstr):
    pids = []
    for lines in os.popen("ps ax | grep " + pstr + " | grep -v grep"):
        line = lines.split()
        pid = line[0]
        pids.append(pid)
       #os.kill(int(pid), signal.SIGKILL)
    print pids

    i = 0
    while i < len(pids):
        rss = get_RSS(pids[i])


if __name__ == '__main__':
    kill_process("usr")