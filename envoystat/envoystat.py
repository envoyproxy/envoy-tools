#!/usr/bin/env python
import argparse
from collections import defaultdict
import datetime
import sys
import signal
import time
import urllib2
import urlparse


GAUGE_NOTATION = "-"
TIMEOUT_SECONDS = 0.1

should_quit = False


def signal_handler(signal, frame):
    global should_quit
    should_quit = True


def main(args):
    signal.signal(signal.SIGINT, signal_handler)

    print datetime.datetime.now().strftime('%Y/%m/%d'), request(args['admin'], '/server_info').read()
    
    header = args['fields']
    formatter = ' '.join('{: >10}' for i in range(len(header)))
    ts = lambda: datetime.datetime.now().strftime('%I:%M:%S %p')

    i = 0
    data_prev, data_now = None, None
    while not should_quit:
        loop_start = time.time()
        if i == 0:
            print ts(), formatter.format(*header)

        if data_prev is None:
            data_prev = get_stats(args['admin'])
        else:
            # compute diff
            data_now = get_stats(args['admin'])

            values = []
            for field in args['fields']:
                final_name = '{}{}'.format(args['prefix'], field.strip(GAUGE_NOTATION))
                if final_name not in data_now:
                    values.append('-')
                if field.endswith(GAUGE_NOTATION):
                    values.append(data_now[final_name])
                else:
                    values.append(data_now[final_name] - data_prev[final_name])
            print ts(), formatter.format(*values)
            data_prev = data_now

        next_loop_delay = args['interval'] - (time.time() - loop_start)
        if next_loop_delay > 0:
            time.sleep(next_loop_delay)
        # repeat header every mod lines
        i = (i + 1) % 25

    # TODO(danielhochman): print a summary
    print

def request(host, path):
    return urllib2.urlopen(
        urlparse.urljoin(host, path),
        timeout=TIMEOUT_SECONDS
    )

def get_stats(host):
    results = defaultdict(int)
    data = request(host, '/stats').readlines()
    for line in data:
        key, value = line.strip().split(': ')
        try:
            results[key] = int(value)
        except ValueError:
            # ignore unparseable values (currently this is only histogram output)
            pass
    return results

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '-a', '--admin', default='http://localhost:9901',
        help="URI to Envoy admin with access to /stats, including scheme",
    )
    parser.add_argument(
        '-p', '--prefix', default='',
        help="Prefix for all requested stats",
    )
    parser.add_argument(
        '-i', '--interval', default=1,
        help="Interval in seconds",
    )
    parser.add_argument(
        '-f', '--fields',
        required=True,
        nargs='*',
        help="field names, excluding the prefix",
    )

    main(vars(parser.parse_args()))
