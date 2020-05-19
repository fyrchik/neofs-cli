# NeoFS CLI

NeoFS CLI is an example of tool that provides all basic interactions with NeoFS. 
It contains commands for container, object, storage group and accounting operations.

## Getting Started 

### Installing NeoFS-CLI
```
$ go get -u github.com/nspcc-dev/neofs-cli
```

### Building NeoFS-CLI

To build NeoFS-CLI run `make build` command:

```
$ git clone github.com/nspcc-dev/neofs-cli.git
$ cd neofs-cli 
$ make build
  ⇒ Ensure vendor: OK
  ⇒ Download requirements: OK
  ⇒ Store vendor localy: OK
  ⇒ Build binary into ./bin/neofs-cli
```

Application will be placed in bin directory

```
$ ./bin/neofs-cli --help
NAME:
   neofs-cli - Example of tool that provides basic interactions with NeoFS

USAGE:
   neofs-cli [global options] command [command options] [arguments...]

VERSION:
   dev (now)

COMMANDS:
   set         set default values for key or host
   object      object manipulation
   sg          storage group manipulation
   container   container manipulation
   withdraw    withdrawals manipulation
   accounting  accounts manipulation
   status      node status info
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --ttl value     request ttl (default: 2)
   --config value  config (default: ".neofs-cli.yml") [$NEOFS_CLI_CONFIG]
   --key value     user private key in hex, wif formats or path to file [$NEOFS_CLI_KEY]
   --host value    host net address [$NEOFS_CLI_ADDRESS]
   --verbose       verbose gRPC connection (default: false)
   --help, -h      show help (default: false)
   --version, -v   print the version (default: false)
   --raw value     use raw request (default: false)

```
### Configuration

User can set up default key and host address values to omit typing it in every 
command. It can be done with environment variables.

```
$ env
...
NEOFS_CLI_KEY=./key
NEOFS_CLI_ADDRESS=fs.nspcc.ru:8080
...
```

You can set up default values within CLI application as well.

```
$ ./bin/neofs-cli set host fs.nspcc.ru:8080
set new value for host: "85.143.219.93:8080"
```

Private key may be represented as path to the file with binary encoded 
private key, 32-byte hex string without `0x` prefix or WIF string.

```
$ ./bin/neofs-cli set key ./key 
set new value for key: "./key"

$ ./bin/neofs-cli set key 1dd37fba80fec4e6a6f13fd708d8dcb3b29def768017052f6c930fa1c5d90bbb
set new value for key: "1dd37fba80fec4e6a6f13fd708d8dcb3b29def768017052f6c930fa1c5d90bbb"

$ ./bin/neofs-cli set key L1ynWYewdiapfZ85bX7hNnhj65jadZcxjmHwN94ST17VrRt6G4Ki
set new value for key: "L1ynWYewdiapfZ85bX7hNnhj65jadZcxjmHwN94ST17VrRt6G4Ki"
```

### Checking available deposit

To perform storage operations like container creation or storage payment user
must have a deposit. To check available deposit run:

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key accounting balance 
Balance info:
- Active balance: 50
```
 
### Container creation

To put object into NeoFS you need to create a container and define storage 
policy. There is an example of new container that stores data in three copies.

**Application will await until container will be accepted by consensus in 
inner ring nodes**

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key container put \
--rule 'SELECT 3 Node'

Container processed: 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG

Trying to wait until container will be accepted on consensus...
...............
Success! Container <7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG> created.
```

### Object operations 

User can upload the object when container is created. You can specify 
user headers that will be put into object's header.

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key object put \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--file ./cat_picture.png \
--user "Nicename"="cat_picture.png"

[./cat_picture.png] Sending header...
[./cat_picture.png] Sending data...
[./cat_picture.png] Object successfully stored
  ID: e35f3596-2cde-4d3e-b57a-752ed687b79a
  CID: 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG
```

All correctly uploaded objects are accessible from CLI application.

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key object get \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--oid e35f3596-2cde-4d3e-b57a-752ed687b79a \
--file ./cat_from_neofs.png

Waiting for data...
Object origin received: e35f3596-2cde-4d3e-b57a-752ed687b79a
receiving chunks: 
Object successfully fetched

$ md5sum cat_from_neofs.png cat_picture.png 
ca940fbc2b7031bd07b510baf397ab01  cat_from_neofs.png
ca940fbc2b7031bd07b510baf397ab01  cat_picture.png
```

You can get object's headers without downloading it from NeoFS.

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key object head \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--oid e35f3596-2cde-4d3e-b57a-752ed687b79a \
--full-headers

Object:
	SystemHeader:
		- ID=7e0b9c6c-aabc-4985-949e-2680e577b48b
		- CID=11111111111111111111111111111111
		- OwnerID=ALYeYC41emF6MrmUMc4a8obEPdgFhq9ran
		- Version=1
		- PayloadLength=1
		- CreatedAt={UnixTime=1 Epoch=1}
	ExtendedHeaders:
		- Type=Link
		  Value={Type=Child ID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=Redirect
		  Value={CID=11111111111111111111111111111111 OID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=UserHeader
		  Value={Key=test_key Val=test_value}
		- Type=Transform
		  Value=Split
		- Type=Tombstone
		  Value=MARKED
		- Type=Verify
		  Value={PublicKey=010203040506 Signature=010203040506}
		- Type=HomoHash
		  Value=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		- Type=PayloadChecksum
		  Value=010203040506
		- Type=Integrity
		  Value={Checksum=010203040506 Signature=010203040506}
		- Type=StorageGroup
		  Value={DataSize=5 Hash=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 Lifetime={Unit=UnixTime Value=555}}
		- Type=PublicKey
		  Value=010203040506
	Payload: []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}
```

You can also search by well known or user defined headers
```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key object search \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
Nicename cat_picture.png

Container ID: Object ID
7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG: e35f3596-2cde-4d3e-b57a-752ed687b79a
```

### Storage group operations

Storage group contains meta information for data audit. If nodes are not 
receiving payments, they will eventually delete objects. To prevent this 
user should create storage group for it's objects.

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key sg put \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--oid e35f3596-2cde-4d3e-b57a-752ed687b79a \
--oid 79ecc573-92c9-4066-8546-96e16e980700 

Storage group successfully stored
        ID: a220d19f-78ca-4574-ac1b-d7b246e929b5
        CID: 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG
```

You can list created storage groups,

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key sg list \ 
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG

Container ID: Object ID
7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG: a220d19f-78ca-4574-ac1b-d7b246e929b5
```

and look for details. Storage group is stored as an object in the container.

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key sg get \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--sgid a220d19f-78ca-4574-ac1b-d7b246e929b5

Object:
	SystemHeader:
		- ID=7e0b9c6c-aabc-4985-949e-2680e577b48b
		- CID=11111111111111111111111111111111
		- OwnerID=ALYeYC41emF6MrmUMc4a8obEPdgFhq9ran
		- Version=1
		- PayloadLength=1
		- CreatedAt={UnixTime=1 Epoch=1}
	ExtendedHeaders:
		- Type=Link
		  Value={Type=Child ID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=Redirect
		  Value={CID=11111111111111111111111111111111 OID=7e0b9c6c-aabc-4985-949e-2680e577b48b}
		- Type=UserHeader
		  Value={Key=test_key Val=test_value}
		- Type=Transform
		  Value=Split
		- Type=Tombstone
		  Value=MARKED
		- Type=Verify
		  Value={PublicKey=010203040506 Signature=010203040506}
		- Type=HomoHash
		  Value=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		- Type=PayloadChecksum
		  Value=010203040506
		- Type=Integrity
		  Value={Checksum=010203040506 Signature=010203040506}
		- Type=StorageGroup
		  Value={DataSize=5 Hash=00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 Lifetime={Unit=UnixTime Value=555}}
		- Type=PublicKey
		  Value=010203040506
	Payload: []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}
```

In the same way you can remove StorageGroup:

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key ./key sg delete \
--cid 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG \
--sgid a220d19f-78ca-4574-ac1b-d7b246e929b5

```

### Status operations

User can request some information about NeoFS node:
- current network map
- current epoch
- current health status
- current metrics
- runtime config

#### Request current network map and epoch

**Network map:**
```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status netmap
{
    "Epoch": 81,
    "NetMap": [{
        "address": "/ip4/165.22.29.184/tcp/8080",
        "pubkey": "AlQKAP3VM2LlSACi8sZjD2tuPXbBqMSU6e+KYUSalbcT",
        "options": ["/Location:Europe/Country:DE/City:Frankfurt", "/Capacity:45", "/Price:20.5"],
        "status": 0
    }, ...]
}
```

**Epoch:**
```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status epoch
81
```

#### Request metrics and health status

**Metrics:**
```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status metrics
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 4.74e-05
go_gc_duration_seconds{quantile="0.25"} 0.0001421
go_gc_duration_seconds{quantile="0.5"} 0.0004501
go_gc_duration_seconds{quantile="0.75"} 0.0033215
go_gc_duration_seconds{quantile="1"} 0.0428934
go_gc_duration_seconds_sum 0.1292919
go_gc_duration_seconds_count 47
...
```

**Health status:**
```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status healthy
Healthy: true
Status: OK
```

#### Request runtime config

*Node must be configured to grant access for certain users. Authentication is made by passed key.*

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status config
{
  "accounting": {
    "events_capacity": 100,
    "log": {
      "balance_lack": false,
      "frs_sum": false
    }
  },
  "app": {
    "name": "neofs-node",
    "version": "0.2.4-4-ge9a43b78(now)"
  },
  "apparitor": {
    "handle_epoch_timeout": "3s",
    "prison_term": 2
  },
  "audit": {
    "epoch_chan_capacity": 100,
    "ir_reward_fee_percents": 1,
    "result_chan_capacity": 100
  },
  ...
}
```

#### Request runtime debug variables

*Node must be configured to grant access for certain users. Authentication is made by passed key.*

```
$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status dump_vars
{...} // variables in ugly json (without formatting)

$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status dump_vars --beauty
// variables in json with formatting
{
  // ...
}
```

#### Change state of specified node:

```
// --state can be `online` or `offline`

$ ./bin/neofs-cli --host fs.nspcc.ru:8080 --key /key status change_state --state offline

DONE

  
```

## License

This project is licensed under the GPLv3 License - 
see the [LICENSE.md](LICENSE.md) file for details
