package main

import (
	"github.com/urfave/cli/v2"
)

type actionName int

const (
	_ actionName = iota
	Global

	Container
	PutContainer
	GetContainer
	DelContainer
	ListContainers

	Object
	GetObject
	PutObject
	DelObject
	HeadObject
	SearchObject
	GetRangeObject
	GetRangeHashObject

	StorageGroup
	GetStorageGroup
	PutStorageGroup
	ListStorageGroups
	DeleteStorageGroup

	Withdraw
	PutWithdraw
	GetWithdraw
	DelWithdraw
	ListWithdraw

	Accounting
	BalanceAccounting

	Status
	GetEpoch
	GetNetmap
	GetMetrics
	GetHealthy
	GetConfig
)

type action struct {
	Flags  []cli.Flag
	Action func(*cli.Context) error
}

var actions = map[actionName]*action{
	Global: {
		Flags: []cli.Flag{ttlF, cfgF, keyFile, hostAddr, verbose},
	},

	// container commands
	Container:      containerAction,
	PutContainer:   putContainerAction,
	GetContainer:   getContainerAction,
	DelContainer:   delContainerAction,
	ListContainers: listContainersAction,

	// object commands
	Object:             objectAction,
	GetObject:          getObjectAction,
	PutObject:          putObjectAction,
	DelObject:          delObjectAction,
	HeadObject:         headObjectAction,
	SearchObject:       searchObjectAction,
	GetRangeObject:     getRangeObjectAction,
	GetRangeHashObject: getRangeHashObjectAction,

	StorageGroup:       sgAction,
	GetStorageGroup:    getSGAction,
	PutStorageGroup:    putSGAction,
	ListStorageGroups:  listSGAction,
	DeleteStorageGroup: delSGAction,

	// withdrawal commands
	Withdraw:     withdrawAction,
	PutWithdraw:  putWithdrawAction,
	GetWithdraw:  getWithdrawAction,
	DelWithdraw:  delWithdrawAction,
	ListWithdraw: listWithdrawAction,

	// accounting commands
	Accounting:        accountingAction,
	BalanceAccounting: getBalanceAction,

	// status commands
	Status:     statusAction,
	GetEpoch:   epochAction,
	GetNetmap:  netmapAction,
	GetMetrics: metricsAction,
	GetHealthy: healthyAction,
	GetConfig:  configAction,
}

func getFlags(name actionName) []cli.Flag {
	return actions[name].Flags
}

func getAction(name actionName) func(*cli.Context) error {
	return actions[name].Action
}
