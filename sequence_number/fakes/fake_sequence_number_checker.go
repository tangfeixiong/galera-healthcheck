// This file was generated by counterfeiter
package fakes

import (
	"net/http"
	"sync"

	"github.com/cloudfoundry-incubator/galera-healthcheck/sequence_number"
)

type FakeSequenceNumberChecker struct {
	CheckStub        func(req *http.Request) (string, error)
	checkMutex       sync.RWMutex
	checkArgsForCall []struct {
		req *http.Request
	}
	checkReturns struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeSequenceNumberChecker) Check(req *http.Request) (string, error) {
	fake.checkMutex.Lock()
	fake.checkArgsForCall = append(fake.checkArgsForCall, struct {
		req *http.Request
	}{req})
	fake.recordInvocation("Check", []interface{}{req})
	fake.checkMutex.Unlock()
	if fake.CheckStub != nil {
		return fake.CheckStub(req)
	} else {
		return fake.checkReturns.result1, fake.checkReturns.result2
	}
}

func (fake *FakeSequenceNumberChecker) CheckCallCount() int {
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	return len(fake.checkArgsForCall)
}

func (fake *FakeSequenceNumberChecker) CheckArgsForCall(i int) *http.Request {
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	return fake.checkArgsForCall[i].req
}

func (fake *FakeSequenceNumberChecker) CheckReturns(result1 string, result2 error) {
	fake.CheckStub = nil
	fake.checkReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeSequenceNumberChecker) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeSequenceNumberChecker) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ sequence_number.SequenceNumberChecker = new(FakeSequenceNumberChecker)
