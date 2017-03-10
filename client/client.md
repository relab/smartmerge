# client

This file describes the different options for running the client:

### General options
```
-alg string
  which algorithms to run (sm | dyna | ssr | cons )
-conf string
  configuration file, a list of address:port, all servers need to be part of this file, also those added by reconfiguration
-initsize int
    	the number of servers in the initial configuration (default 1)
-mode string
  user starts an interactive client (default)
  bench benchmarking, for clients performing mutliple reads or writes
  exp for clients performing reconfigurations
-id int
  client id
-nclients int
  number of clients, default 1
```
The client will use an initial configuration containing the `-initsize` first servers in the configuration file.

If `-nclient` is specified, several clients with consecutive ids, starting with the specified `-id` will be started.

###Configuration provider

This option determines which processes are contacted on performing an rpc.
```
-cprov string
  which configuration provider: (normal | thrifty | norecontact )  (default "normal")
```
* normal: contact all servers in a configuration and wait for replies from a quorum
* thrifty: contact only the servers in one quorum
* norecontact: For algorithms sm (SM-Store) and cons (Rambo) this option activates single contact mode

When using `thrifty, the quorum of servers contacted is determined by the clients id modulo the number of servers in the configuration.
To ensure even load distribution use consecutive client ids.

See the paper for an explanation of *single contact mode* `norecontact`.
  
###Performing reads and writes
Single reads and writes can be performed in the interactive `user` mode.

To perform multiple reads/writes use `-mode bench`.

The following options start a client performing multiple reads/ writes.
A fixed number of operations can be performed using `-reads` or `-writes`, while
`-contR` and `-contW` start clients continously performing reads/writes, until termination signal is received. 
For *writes*, `size` can be used to determine the size of the value written. 
For *reads*, `regular`can be used to perform regular reads, that do omit writing back value.
```
-reads int
    	number of reads to be performed.
-contR
    	continuously read
-regular
    	do only regular reads
-writes int
    	number of writes to be performed.
-contW
    	continuously write
 -size int
    	number of bytes for value. (default 16)
  ```

###Performing reconfigurations
Single reconfigurations can be performed in the interactive `user` mode.

To perform multiple reconfigurations use `-mode exp`.
All servers added in reconfigurations must be part of the configuration file.

```
-nclients int
    	the number of clients (default 1)
-rm
    	remove nclients servers concurrently.
-add
    	remove nclients servers concurrently.
-repl
    	replace nclient many servers concurrently.
-cont 
      continously perform reconfigurations
```

The `-nclients` option can be used to start several clients that will concurrently initialize reconfigurations.
`-nclients=4 -id=2` will start 4 clients with ids 2, 3, 4, and 5.
`-rm` and `-add` can be used to have the clients concurrently propose a reconfiguration each adding/removing one server.
`-repl -id=2` starts a client that replaces the second server in the initial configuration with the second last server 
in the configuration file. `-cont` can be used to start a server that continously performs the same reconfiguration, 
e.g. replacing a server with a different one, if this reconfiguration is possible.

### Processing logs
Running the client in benchmarking or experiment mode will produce `.elog` files containing latency or throughput data.
The directory `elog/util/efmt`contains a rudimentary program to process these files.
```
cd elog/util/efmt
go install efmt
```
The `efmt` command takes processes latencies, according to how many round trips were needed to complete the operation. For SmartMerge or Rambo run  `efmt -normal=2 -file=myFile.elog`. You can also supply a list of several `.elog` files.
