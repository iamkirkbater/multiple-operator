package controller

import (
	"github.com/iamkirkbater/multiple-operator/pkg/controller/spike"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, spike.Add)
}
