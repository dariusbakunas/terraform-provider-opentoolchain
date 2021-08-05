package opentoolchain

import (
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestMakeEnvPatch(t *testing.T) {
	testcases := []struct {
		currentEnv    []oc.EnvProperty
		textEnv       interface{}
		secretEnv     interface{}
		deletedKeys   interface{}
		originalProps interface{}
		expected      []oc.EnvProperty
	}{
		{
			currentEnv: []oc.EnvProperty{
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("true"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("sdoidhjsofjsodjfi"), Type: getStringPtr("SECURE")},
				{Name: getStringPtr("DEL_TEXT"), Value: getStringPtr("text"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("DEL_SECRET"), Value: getStringPtr("secret"), Type: getStringPtr("SECURE")},
			},
			textEnv: map[string]interface{}{
				"DISABLE_DEBUG_LOGGING": "false",
				"NEW_PROP":              "some text",
			},
			secretEnv: map[string]interface{}{
				"ASOCAPIKEYSECRET": "new secret value",
				"NEW_SECRET":       "some secret",
			},
			deletedKeys:   []interface{}{"DEL_TEXT", "DEL_SECRET"},
			originalProps: []interface{}{},
			expected: []oc.EnvProperty{
				{Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("new secret value"), Type: getStringPtr("SECURE")},
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("false"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("NEW_PROP"), Value: getStringPtr("some text"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("NEW_SECRET"), Value: getStringPtr("some secret"), Type: getStringPtr("SECURE")},
			},
		},
		{
			currentEnv: []oc.EnvProperty{
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("true"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("new secret"), Type: getStringPtr("SECURE")},
				{Name: getStringPtr("NEW_PROP"), Value: getStringPtr("some text"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("SOME_SECRET"), Value: getStringPtr("new secret"), Type: getStringPtr("SECURE")},
			},
			textEnv: map[string]interface{}{
				// test removal of "DISABLE_DEBUG_LOGGING:true" property, should be restored to false
				"NEW_PROP": "some text", // should stay the same
			},
			secretEnv: map[string]interface{}{
				// test removal of "SOME_SECRET:new secret", should be restored to "original secret"
				"ASOCAPIKEYSECRET": "new secret", // should stay the same
			},
			deletedKeys: []interface{}{}, // test removal of DEL_TEXT, DEL_SECRET, should be restored to originals
			originalProps: []interface{}{
				map[string]interface{}{"name": "DISABLE_DEBUG_LOGGING", "value": "false", "type": "TEXT"},
				map[string]interface{}{"name": "DEL_TEXT", "value": "original text", "type": "TEXT"},
				map[string]interface{}{"name": "DEL_SECRET", "value": "original secret", "type": "SECURE"},
				map[string]interface{}{"name": "ASOCAPIKEYSECRET", "value": "original secret", "type": "SECURE"},
				map[string]interface{}{"name": "NEW_PROP", "value": "original text", "type": "TEXT"},
				map[string]interface{}{"name": "SOME_SECRET", "value": "original secret", "type": "SECURE"},
			},
			expected: []oc.EnvProperty{
				{Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("new secret"), Type: getStringPtr("SECURE")}, // we're still overriding
				{Name: getStringPtr("DEL_SECRET"), Value: getStringPtr("original secret"), Type: getStringPtr("SECURE")},  // restored to original
				{Name: getStringPtr("DEL_TEXT"), Value: getStringPtr("original text"), Type: getStringPtr("TEXT")},        // restored to original
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("false"), Type: getStringPtr("TEXT")},   // restored to original
				{Name: getStringPtr("NEW_PROP"), Value: getStringPtr("some text"), Type: getStringPtr("TEXT")},            // should stay the same
				{Name: getStringPtr("SOME_SECRET"), Value: getStringPtr("original secret"), Type: getStringPtr("SECURE")}, // restored to original
			},
		},
	}

	for _, c := range testcases {
		actual := makeEnvPatch(c.currentEnv, c.textEnv, c.secretEnv, c.deletedKeys, c.originalProps)

		sort.Slice(actual, func(i, j int) bool {
			return *actual[i].Name < *actual[j].Name
		})

		assert.Equal(t, c.expected, actual)
	}
}

func TestKeepOriginalProps(t *testing.T) {
	testcases := []struct {
		currentEnv       []oc.EnvProperty
		textEnv          interface{}
		secretEnv        interface{}
		deletedKeys      interface{}
		expectedOriginal []interface{}
		expectedNew      []interface{}
	}{
		{
			currentEnv: []oc.EnvProperty{
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("true"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("sdoidhjsofjsodjfi"), Type: getStringPtr("SECURE")},
				{Name: getStringPtr("DEL_TEXT"), Value: getStringPtr("text"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("SOME_SECRET"), Value: getStringPtr("secret text"), Type: getStringPtr("SECURE")},
			},
			textEnv: map[string]interface{}{
				"DISABLE_DEBUG_LOGGING": "false",
				"NEW_PROP":              "some text",
			},
			secretEnv: map[string]interface{}{
				"SOME_SECRET": "some secret",
				"NEW_SECRET":  "new secret",
			},
			deletedKeys: []interface{}{"DEL_TEXT", "DEL_SECRET"},
			expectedOriginal: []interface{}{
				map[string]interface{}{"name": "DEL_TEXT", "value": "text", "type": "TEXT"},
				map[string]interface{}{"name": "DISABLE_DEBUG_LOGGING", "value": "true", "type": "TEXT"},
				map[string]interface{}{"name": "SOME_SECRET", "value": "secret text", "type": "SECURE"},
			},
			expectedNew: []interface{}{"NEW_PROP", "NEW_SECRET", "DEL_SECRET"},
		},
	}

	for _, c := range testcases {
		actualOriginal, actualNew := createOriginalProps(c.currentEnv, c.textEnv, c.secretEnv, c.deletedKeys)
		// no need to sort here, method already sorts final result, which we should also be testing
		assert.Equal(t, c.expectedOriginal, actualOriginal)
		assert.Equal(t, c.expectedNew, actualNew)
	}
}

func TestUpdateOriginalProps(t *testing.T) {
	testcases := []struct {
		currentEnv    []oc.EnvProperty
		textEnv       interface{}
		secretEnv     interface{}
		deletedKeys   interface{}
		newKeys       interface{}
		originalProps interface{}
		expected      []interface{}
	}{
		{
			currentEnv: []oc.EnvProperty{
				{Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("true"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("NEW_TEXT"), Value: getStringPtr("original text"), Type: getStringPtr("TEXT")},
				{Name: getStringPtr("NEW_SECRET"), Value: getStringPtr("original secret"), Type: getStringPtr("SECURE")},
				{Name: getStringPtr("DELETED_TEXT"), Value: getStringPtr("original text"), Type: getStringPtr("TEXT")},
			},
			textEnv: map[string]interface{}{
				"DISABLE_DEBUG_LOGGING": "false",
				"NEW_TEXT":              "new text",
			},
			secretEnv: map[string]interface{}{
				"NEW_SECRET": "new secret",
			},
			deletedKeys: []interface{}{"DELETED_TEXT"},
			newKeys:     []interface{}{},
			originalProps: []interface{}{
				map[string]interface{}{"name": "DISABLE_DEBUG_LOGGING", "value": "true", "type": "TEXT"},
			},
			expected: []interface{}{
				map[string]interface{}{"name": "DELETED_TEXT", "value": "original text", "type": "TEXT"},
				map[string]interface{}{"name": "DISABLE_DEBUG_LOGGING", "value": "true", "type": "TEXT"},
				map[string]interface{}{"name": "NEW_SECRET", "value": "original secret", "type": "SECURE"},
				map[string]interface{}{"name": "NEW_TEXT", "value": "original text", "type": "TEXT"},
			},
		},
	}

	for _, c := range testcases {
		// TODO: test update to newKeys
		actual, _ := updateOriginalProps(c.currentEnv, c.textEnv, c.secretEnv, c.deletedKeys, c.newKeys, c.originalProps)
		// no need to sort here, method already sorts final result, which we should also be testing
		assert.Equal(t, c.expected, actual)
	}
}
