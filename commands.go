package main

import "github.com/urfave/cli/v2"

func commands() cli.Commands {
	return cli.Commands{
		{
			Name:      "set",
			Usage:     "set default values for key or host",
			UsageText: "set <mode:key|host> <value>",
			Subcommands: cli.Commands{
				{
					Name:        "key",
					Usage:       "set default value for key",
					UsageText:   "set key <value>",
					Description: "set user default value for key",
					Action:      setCommand(KeyMode),
				},
				{
					Name:        "host",
					Usage:       "set default value for host",
					UsageText:   "set host <value>",
					Description: "set user default value for host",
					Action:      setCommand(HostMode),
				},
			},
		},
		{
			Name:      "object",
			Usage:     "object manipulation",
			UsageText: "object <subcommand> [arguments...]",
			Flags:     getFlags(Object),
			Subcommands: cli.Commands{
				{
					Name:  "put",
					Usage: "put object into container",
					UsageText: "put --cid <cid> --file </path/to/file> " +
						"[--perm <permissions>] [--verify] [--user key1=value1 ...]",
					Description: "put user data into container",
					Flags:       getFlags(PutObject),
					Action:      getAction(PutObject),
				},
				{
					Name:        "get",
					Usage:       "get object from container",
					UsageText:   "get --cid <cid> --oid <oid> --file ./my-file [--perm <permissions>] [--raw]",
					Description: "get file from network",
					Flags:       getFlags(GetObject),
					Action:      getAction(GetObject),
				},
				{
					Name:        "delete",
					Usage:       "delete object from container",
					UsageText:   "delete --cid <cid> --oid <oid>",
					Description: "delete file from network",
					Flags:       getFlags(DelObject),
					Action:      getAction(DelObject),
				},
				{
					Name:        "head",
					Usage:       "get object header from container",
					UsageText:   "head --cid <cid> --oid <oid> [--full-headers] [--raw]",
					Description: "retrieve object metadata",
					Flags:       getFlags(HeadObject),
					Action:      getAction(HeadObject),
				},
				{
					Name:        "search",
					Usage:       "perform search query within container",
					UsageText:   "search --cid <cid> [<key1> <query1> [<key2> <query2>...]]",
					Description: "search object by headers",
					Flags:       getFlags(SearchObject),
					Action:      getAction(SearchObject),
				},
				{
					Name:      "get-range",
					Usage:     "get data of the object payload ranges from container",
					UsageText: "get-range --cid <cid> --oid <oid> [<offset1>:<length1> [...]]",
					Flags:     getFlags(GetRangeObject),
					Action:    getAction(GetRangeObject),
				},
				{
					Name:      "get-range-hash",
					Usage:     "get homomorphic hash of the object payload ranges from container",
					UsageText: "get-range-hash --cid <cid> --oid <oid> [--verify --file </path/to/file>] [--salt <hex>] [<offset1>:<length1> [...]]",
					Flags:     getFlags(GetRangeHashObject),
					Action:    getAction(GetRangeHashObject),
				},
			},
		},
		{
			Name:      "sg",
			Usage:     "storage group manipulation",
			UsageText: "sg <subcommand> [arguments...]",
			Flags:     getFlags(StorageGroup),
			Subcommands: cli.Commands{
				{
					Name:        "put",
					Usage:       "put storage group in system",
					UsageText:   "put --cid <cid> --sgid <uuid>...",
					Description: "put new storage group",
					Flags:       getFlags(PutStorageGroup),
					Action:      getAction(PutStorageGroup),
				},
				{
					Name:        "get",
					Usage:       "get storage group from the system",
					UsageText:   "get --cid <cid> --sgid <sgid> ",
					Description: "get information about existing storage group",
					Flags:       getFlags(GetStorageGroup),
					Action:      getAction(GetStorageGroup),
				},
				{
					Name:        "list",
					Usage:       "list available storage groups",
					UsageText:   "list --cid <cid> ",
					Description: "list user's storage groups",
					Flags:       getFlags(ListStorageGroups),
					Action:      getAction(ListStorageGroups),
				},
				{
					Name:        "delete",
					Usage:       "delete storage group from the system",
					UsageText:   "delete --cid <cid> --sgid <sgid>",
					Description: "delete user's storage group",
					Flags:       getFlags(DeleteStorageGroup),
					Action:      getAction(DeleteStorageGroup),
				},
			},
		},
		{
			Name:      "container",
			Usage:     "container manipulation",
			UsageText: "container <subcommand> [arguments...]",
			Flags:     getFlags(Container),
			Subcommands: cli.Commands{
				{
					Name:        "put",
					Usage:       "put container",
					UsageText:   "put --rule 'SELECT 3 Node FILTER State NE IR' [--cap <cap-in-GB>]",
					Description: "put container into network",
					Flags:       getFlags(PutContainer),
					Action:      getAction(PutContainer),
				},
				{
					Name:        "get",
					Usage:       "get container",
					UsageText:   "get --cid <cid>",
					Description: "fetch container info from network",
					Flags:       getFlags(GetContainer),
					Action:      getAction(GetContainer),
				},
				{
					Name:        "delete",
					Usage:       "delete container",
					UsageText:   "delete --cid <cid>",
					Description: "delete container from network",
					Flags:       getFlags(DelContainer),
					Action:      getAction(DelContainer),
				},
				{
					Name:        "list",
					Usage:       "list user containers",
					UsageText:   "list",
					Description: "list user containers which are stored in network",
					Flags:       getFlags(ListContainers),
					Action:      getAction(ListContainers),
				},
			},
		},
		{
			Name:      "withdraw",
			Usage:     "withdrawals manipulation",
			UsageText: "withdraw <subcommand> [arguments...]",
			Flags:     getFlags(Withdraw),
			Subcommands: cli.Commands{
				{
					Name:        "put",
					Usage:       "create request for withdrawal",
					UsageText:   "put --amount <amount> --height <height>",
					Description: "put user data into container",
					Flags:       getFlags(PutWithdraw),
					Action:      getAction(PutWithdraw),
				},
				{
					Name:        "get",
					Usage:       "get withdrawal",
					UsageText:   "get --wid <wid>",
					Description: "fetch withdrawal info from network",
					Flags:       getFlags(GetWithdraw),
					Action:      getAction(GetWithdraw),
				},
				{
					Name:        "delete",
					Usage:       "delete withdrawal",
					UsageText:   "delete --wid <wid>",
					Description: "delete withdrawal from network",
					Flags:       getFlags(DelWithdraw),
					Action:      getAction(DelWithdraw),
				},
				{
					Name:        "list",
					Usage:       "list user withdrawals",
					UsageText:   "list",
					Description: "list user withdrawals which are stored in network",
					Flags:       getFlags(ListWithdraw),
					Action:      getAction(ListWithdraw),
				},
			},
		},
		{
			Name:      "accounting",
			Usage:     "accounts manipulation",
			UsageText: "accounting <subcommand> [arguments...]",
			Flags:     getFlags(Accounting),
			Subcommands: cli.Commands{
				{
					Name:        "balance",
					Usage:       "get user account balance",
					UsageText:   "balance",
					Description: "get user balance from network",
					Flags:       getFlags(BalanceAccounting),
					Action:      getAction(BalanceAccounting),
				},
			},
		},
		{
			Name:      "status",
			Usage:     "node status info",
			UsageText: "status <subcommand> [arguments...]",
			Flags:     getFlags(Status),
			Subcommands: cli.Commands{
				{
					Name:        "netmap",
					Usage:       "get json copy of the node's active network map",
					UsageText:   "netmap",
					Description: "get json copy of the node's active network map",
					Flags:       getFlags(GetNetmap),
					Action:      getAction(GetNetmap),
				},
				{
					Name:        "epoch",
					Usage:       "get current epoch of the node",
					UsageText:   "epoch",
					Description: "get current epoch of the node",
					Flags:       getFlags(GetEpoch),
					Action:      getAction(GetEpoch),
				},
				{
					Name:        "metrics",
					Usage:       "get metrics of the node",
					UsageText:   "node state metrics",
					Description: "get metrics of the node",
					Flags:       getFlags(GetMetrics),
					Action:      getAction(GetMetrics),
				},
				{
					Name:        "healthy",
					Usage:       "health check of the node",
					UsageText:   "node health checker",
					Description: "health check of the node",
					Flags:       getFlags(GetHealthy),
					Action:      getAction(GetHealthy),
				},
				{
					Name:        "config",
					Usage:       "dump config of specified node",
					UsageText:   "neofs-cli --host <host:port> --key <key:path|hex|wif> status config",
					Description: "allows dumping runtime config of specified node",
					Flags:       getFlags(GetConfig),
					Action:      getAction(GetConfig),
				},
			},
		},
	}
}
