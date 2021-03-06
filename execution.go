package ib

import (
	"time"
)

// This file ports IB API Execution.java. Please preserve declaration order.

type Execution struct {
	OrderId      int64
	ClientId     int64
	ExecId       string
	Time         time.Time
	AccountCode  string
	Exchange     string
	Side         string
	Shares       int64
	Price        float64
	PermId       int64
	Liquidation  int64
	CumQty       int64
	AveragePrice float64
	OrderRef     string
	EVRule       string
	EVMultiplier float64
}
