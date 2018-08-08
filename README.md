
Redigo Stage 1:
Mono-machine version redis, with zset(skiplist version), list(linkedlist), kv, hash and set.
The string is the original go str.

The stage one version should have a simple version server and client with simple command line window as the same as redis
(The client should embeded in the go program socket programming)

progress:
0. core frame
1. kv
2. client and server, command system
3. list
4. set
5. hash
6. zset

Redigo Stage 2:
LRU
log
idle time
persistence rdb
persistence aof


Redigo Stage 3:
ziplist
intset
quicklist