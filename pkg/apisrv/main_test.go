package apisrv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Juniper/contrail/pkg/testutil/integration"
)

var server *integration.APIServer

func TestMain(m *testing.M) {
	integration.TestMain(m, &server)
}

func RunTest(t *testing.T, file string) {
	testScenario, err := integration.LoadTest(file, nil)
	assert.NoError(t, err, "failed to load test data")
	integration.RunCleanTestScenario(t, testScenario, server)
}

func RunTestTemplate(t *testing.T, file string, context map[string]interface{}) {
	var testScenario integration.TestScenario
	err := integration.LoadTestScenario(&testScenario, file, context)
	assert.NoError(t, err, "failed to load test data")
	integration.RunCleanTestScenario(t, &testScenario, server)
}
