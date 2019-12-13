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

System headers:
  Object ID   : e35f3596-2cde-4d3e-b57a-752ed687b79a
  Owner ID    : AK2nJJpJr6o664CWJKi1QRXjqeic2zRp8y
  Container ID: 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG
  Payload Size: 3.1M
  Version     : 1
  Created at  : epoch #13, 2019-11-07 17:42:57 +0300 MSK
Other headers:
  UserHeader:<Key:"Nicename" Value:"cat_picture.png" > 
  Verify:<PublicKey:"\002{...}" >
  HomoHash:... 
  PayloadChecksum:"..."
  Integrity:<HeadersChecksum:"..." ChecksumSignature:"..." >
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

System headers:
  Object ID   : a220d19f-78ca-4574-ac1b-d7b246e929b5
  Owner ID    : AK2nJJpJr6o664CWJKi1QRXjqeic2zRp8y
  Container ID: 7Gi7c1WmyKxEW3JwqEETupNoQ7rAb1CSQYxdPirXLwaG
  Payload Size: 0B
  Version     : 1
  Created at  : epoch #16, 2019-11-07 17:59:57 +0300 MSK
Other headers:
  Link:<type:StorageGroup ID:e35f3596-2cde-4d3e-b57a-752ed687b79a > 
  Link:<type:StorageGroup ID:79ecc573-92c9-4066-8546-96e16e980700 > 
  StorageGroup:<ValidationDataSize:6500758 ValidationHash:... > 
  Verify: ...
  HomoHash: ... 
  PayloadChecksum: ...
  Integrity: ...
```

## License

This project is licensed under the GPLv3 License - 
see the [LICENSE.md](LICENSE.md) file for details
