# gnt
network tools written in go

- [x] ping
- [ ] traceroute
- [ ] tcpdump
- [ ] curl
- [ ] telnet
- [ ] nc

## build

build libpcap
```bash
export PCAPV=1.9.1
wget http://www.tcpdump.org/release/libpcap-$PCAPV.tar.gz && \
    tar xvf libpcap-$PCAPV.tar.gz && \
    cd libpcap-$PCAPV && \
    ./configure --enable-dbus=no --enable-shared=no && \
    make 

```

```bash
# cd in project dir
make build
```

## ping

## traceroute
- ref doc: https://linux.die.net/man/8/traceroute
- source code: https://github.com/openbsd/src/blob/master/usr.sbin/traceroute/traceroute.c
