package templates

import (
	"fmt"
	"testing"
)

func TestTemplates(t *testing.T) {

	tf := NewTextField("label", "name")
	tf.CanEdit = false

	out := tf.Render()

	fmt.Printf("out=%#v\n", out)
}
