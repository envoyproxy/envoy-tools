# envoystat

envoystat displays information from the envoy stats endpoint at an interval. This is useful for
monitoring the behavior of a system in real-time.

```
$ python envoystat.py 
2018/04/04 envoy 19745a5e3d267b8621f6b0740250bbe04af3fadc/1.7.0-dev/Clean/RELEASE live 66476 66476 0

03:57:30 PM cx_active- rq_active-     rq_2xx     rq_4xx     rq_5xx
03:57:31 PM        371          3         94          0          4
03:57:32 PM        371          5         66          3          5
03:57:33 PM        371          1         99          0          6
03:57:34 PM        371          0         58          9          2
03:57:35 PM        371          2         51          0          1
^C
```

The `-` suffix denotes a gauge. All other values are assumed to be counters. At each interval, the
script will display the difference for each counter and the current value of each gauge passed in
the `--fields` parameter.

# Dependencies

envoystat has no dependencies other than Python 2.

# Installation

Run directly with `python envoystat.py` or symlink it into the system path. In the future, envoystat
will be a Python module installable via pip.
  
# Usage

`envoystat [--admin <uri>] [--prefix <prefix>] [--fields <field ...>] [--interval <interval>]`

`-a, --admin`
> Admin URI. The script will request the `/stats` endpoint from this URI. Defaults to `http://localhost:9901`.

`-p, --prefix`
> Prefix for all given fields. The field names will be appended to the prefix. Defaults to an empty string.

`-f, --fields`
> List of fields for the given prefix. The field names will be appended to the prefix and used for header display.

`-i, --interval`
> Interval in seconds. Defaults to 1 second.


# Example
If you want to monitor the http_connection_manager with stat_prefix `ingress_http` for 5xx,
normally you might do the following:

```
 $ watch 'curl -s http://localhost:9901/stats | grep ingress_http'
http.ingress_http.buffer.rq_timeout: 0
http.ingress_http.downstream_cx_active: 373
http.ingress_http.downstream_cx_destroy: 1721
http.ingress_http.downstream_cx_destroy_active_rq: 0
http.ingress_http.downstream_cx_destroy_local: 23
...
```

Instead, using envoystat, you can provide a common prefix, `-p http.ingress_http.downstream_`, and
then list the fields you want see on a per-interval basis. If we're interested in
`http.ingress_http.downstream_rq_2xx` and `http.ingress_http.downstream_rq_5xx`, add
`-f rq_2xx rq_5xx`. This results in

```
$ python envoystat.py -p http.ingress_http.downstream_ -f rq_2xx rq_5xx
2018/04/04 envoy 19745a5e3d267b8621f6b0740250bbe04af3fadc/1.7.0-dev/Clean/RELEASE live 69245 69245 0

04:43:39 PM     rq_2xx     rq_5xx
04:43:40 PM         64          4
04:43:41 PM         71          2
04:43:42 PM         64         10
...
```

# Beta Roadmap

* Make more generic and allow users to discover invocations using the tool itself.
* More robust argument validation and error handling.
* Create Python module and upload to PyPI for easy end-user installation.
* Consume a config file from a known location or environment variables.
* Tests.
