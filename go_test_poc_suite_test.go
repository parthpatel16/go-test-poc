package go_test_poc_test_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGoTestPoc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoTestPoc Suite")
}
