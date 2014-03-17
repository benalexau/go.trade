package trade

import (
	"errors"
	"flag"
	"reflect"
	"testing"
	"time"
)

func (engine *Engine) expect(t *testing.T, seconds int, ch chan Reply, expected []IncomingMessageId) (Reply, error) {
	for {
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			return nil, errors.New("Timeout waiting")
		case v := <-ch:
			if v.code() == 0 {
				t.Fatalf("don't know message '%v'", v)
			}
			for _, code := range expected {
				if v.code() == code {
					return v, nil
				}
			}
			// wrong message received
			t.Logf("received message '%v' of type '%v'\n",
				v, reflect.ValueOf(v).Type())
		}
	}

	return nil, nil
}

// private variable for mantaining engine reuse in test
// use TestEngine instead of this
var testEngine *Engine
var noEngineReuse = flag.Bool("no-engine-reuse", false,
	"Don't keep reusing the engine; each test case gets its own engine.")

// Engine for test reuse.
//
// Unless the test runner is passed the -no-engine-reuse flag, this will keep
// reusing the same engine.
func NewTestEngine(t *testing.T) *Engine {

	if testEngine == nil {

		engine, err := NewEngine()

		if err != nil {
			t.Fatalf("cannot connect engine: %s", err)
		}

		if *noEngineReuse {
			t.Log("created new engine, no reuse")
			return engine
		} else {
			t.Log("created engine for reuse")
			testEngine = engine
			return engine
		}
	}

	if testEngine.State() != EngineReady {
		t.Fatalf("engine (client ID %d) not ready (did a prior test Stop() rather than ConditionalStop() ?)", testEngine.client)
	}

	t.Log("reusing engine")
	return testEngine
}

// Will actually do a stop only if the flag -no-engine-reuse is active
func (e *Engine) ConditionalStop(t *testing.T) {
	if *noEngineReuse {
		t.Log("no engine reuse, stopping engine")
		e.Stop()
	}
}

func TestConnect(t *testing.T) {
	engine, err := NewEngine()

	if err != nil {
		t.Fatalf("cannot connect engine: %s", err)
	}

	defer engine.Stop()

	if engine.State() != EngineReady {
		t.Fatalf("engine is not ready")
	}

	if engine.serverTime.IsZero() {
		t.Fatalf("server time not provided")
	}

	var states chan EngineState = make(chan EngineState)
	engine.SubscribeState(states)

	// stop the engine in 100 ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		engine.Stop()
	}()

	newState := <-states

	if newState != EngineExitNormal {
		t.Fatalf("engine state change error")
	}

	err = engine.FatalError()
	if err != nil {
		t.Fatalf("engine reported an error: %v", err)
	}
}

func TestMarketData(t *testing.T) {
	engine := NewTestEngine(t)

	defer engine.ConditionalStop(t)

	req1 := &RequestMarketData{
		Contract: Contract{
			Symbol:       "AUD",
			SecurityType: "CASH",
			Exchange:     "IDEALPRO",
			Currency:     "USD",
		},
	}

	id := engine.NextRequestId()
	req1.SetId(id)
	ch := make(chan Reply)
	engine.Subscribe(ch, id)

	if err := engine.Send(req1); err != nil {
		t.Fatalf("client %d: cannot send market data request: %s", engine.ClientId(), err)
	}

	rep1, err := engine.expect(t, 30, ch, []IncomingMessageId{mTickPrice, mTickSize})
	logreply(t, rep1, err)

	if err != nil {
		t.Fatalf("client %d: cannot receive market data: %s", engine.ClientId(), err)
	}

	if err := engine.Send(&CancelMarketData{id}); err != nil {
		t.Fatalf("client %d: cannot send cancel request: %s", engine.ClientId(), err)
	}

	engine.Unsubscribe(ch, id)
}

func TestContractDetails(t *testing.T) {
	engine := NewTestEngine(t)

	defer engine.ConditionalStop(t)

	req1 := &RequestContractData{
		Contract: Contract{
			Symbol:       "AAPL",
			SecurityType: "STK",
			Exchange:     "SMART",
			Currency:     "USD",
		},
	}

	id := engine.NextRequestId()
	req1.SetId(id)
	ch := make(chan Reply)
	engine.Subscribe(ch, id)
	defer engine.Unsubscribe(ch, id)

	if err := engine.Send(req1); err != nil {
		t.Fatalf("client %d: cannot send contract data request: %s", engine.ClientId(), err)
	}

	rep1, err := engine.expect(t, 30, ch, []IncomingMessageId{mContractData})
	logreply(t, rep1, err)

	if err != nil {
		t.Fatalf("client %d: cannot receive contract details: %s", engine.ClientId(), err)
	}

	rep2, err := engine.expect(t, 30, ch, []IncomingMessageId{mContractDataEnd})
	logreply(t, rep2, err)

	if err != nil {
		t.Fatalf("client %d: cannot receive end of contract details: %s", engine.ClientId(), err)
	}
}

func TestOptionChainRequest(t *testing.T) {
	engine := NewTestEngine(t)

	defer engine.ConditionalStop(t)

	req1 := &RequestContractData{
		Contract: Contract{
			Symbol:       "AAPL",
			SecurityType: "OPT",
			Exchange:     "SMART",
			Currency:     "USD",
		},
	}

	id := engine.NextRequestId()
	req1.SetId(id)
	ch := make(chan Reply)
	engine.Subscribe(ch, id)
	defer engine.Unsubscribe(ch, id)

	if err := engine.Send(req1); err != nil {
		t.Fatalf("cannot send contract data request: %s", err)
	}

	rep1, err := engine.expect(t, 30, ch, []IncomingMessageId{mContractDataEnd})
	logreply(t, rep1, err)

	if err != nil {
		t.Fatalf("cannot receive contract details: %v", err)
	}
}

func logreply(t *testing.T, reply Reply, err error) {
	if reply == nil {
		t.Logf("received reply nil")
	} else {
		t.Logf("received reply '%v' of type %v", reply, reflect.ValueOf(reply).Type())
	}
	if err != nil {
		t.Logf(" (error: '%v')", err)
	}
	t.Logf("\n")
}
