# Description

A command line tool to check multiple server regularly with different checks like DNS resolution or ping reachability.

# Check Plugins

* DNS
* Ping

# Basic Usage

After build execute the binary with the wanted check plugin and the servers that you want to request:

```shell
./parallel-check -p DNS -d example.com  8.8.8.8 8.8.4.4 208.67.222.222 208.67.220.220 127.0.0.1
|   DNS SERVER   | SUCCESS | ERRORS | ERROR % |   LAST   | AVERAGE  |   BEST   |  WORST   |       QUERY HISTORY        |
|----------------|---------|--------|---------|----------|----------|----------|----------|----------------------------|
| 8.8.8.8        |      26 |      0 | 0.00%   | 21.52 ms | 24.97 ms | 20.45 ms | 29.45 ms | -+-+---+--*----+-++++----- |
| 8.8.4.4        |      26 |      0 | 0.00%   | 25.53 ms | 25.93 ms | 20.20 ms | 28.36 ms | +--++-++-+++-+--++++--+-+- |
| 208.67.222.222 |      26 |      0 | 0.00%   | 29.36 ms | 26.74 ms | 21.52 ms | 32.59 ms | +-++++#-+-#+*+*---#-++---* |
| 208.67.220.220 |      26 |      0 | 0.00%   | 29.36 ms | 28.93 ms | 23.10 ms | 32.51 ms | --+*++#*++++++****#*+++#** |
| 127.0.0.1      |       0 |     26 | 100.00% | 0.00 ms  | 0.00 ms  | 0.00 ms  | 0.00 ms  | ?????????????????????????? |

  Scale: . < 19ms - < 26ms + < 29ms * < 30ms # < 32ms
  Query History: 59 Requests / ~1m1s
  Timeout: 1s | Delay: 1s
```

The tool has a help when you call it without arguments:

```
Usage: parallel-check.exe [<arguments>] <IP> [<IP> ...]

This tool do execute a check with the given address in a regular interval and prints
the results to the terminal.

Interactive Keyboard Shortcuts:
  Q: Quit
  P: Pause
  R: Reset
  Arrow Key Up: Increase Wait Time between Checks
  Arrow Key Down: Decrease Wait Time
  Arrow Key Left: Decrease Timeout
  Arrow Key Right: Increase Timeout

Arguments:
  -4    use IPv4
  -6    use IPv6
  -c count
        exit after count tests
  -d domain
        dns check: domain that should be queried (default "example.com")
  -p string
        shorthand for --plugin (default "dns")
  -plugin string
        which check plugin should be used. Available: [dns ping] (default "dns")
  -t duration
        timeout for checks (prefix duration with ms or s) (default 1s)
  -w duration
        delay between two checks (prefix duration with ms or s) (default 1s)
```