package opentoolchain

import (
    oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
    "github.com/stretchr/testify/assert"
    "sort"
    "testing"
)

func TestMakeEnvPatch(t *testing.T) {
    testcases := []struct {
        currentEnv []oc.EnvProperty
        textEnv interface{}
        secretEnv interface{}
        deletedKeys interface{}
        expected []oc.EnvProperty
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
                "NEW_PROP": "some text",
            },
            secretEnv: map[string]interface{}{
                "ASOCAPIKEYSECRET": "new secret value",
                "NEW_SECRET": "some secret",
            },
            deletedKeys: []interface{}{"DEL_TEXT", "DEL_SECRET"},
            expected: []oc.EnvProperty{
                {Name: getStringPtr("ASOCAPIKEYSECRET"), Value: getStringPtr("new secret value"), Type: getStringPtr("SECURE")},
                {Name: getStringPtr("DISABLE_DEBUG_LOGGING"), Value: getStringPtr("false"), Type: getStringPtr("TEXT")},
                {Name: getStringPtr("NEW_PROP"), Value: getStringPtr("some text"), Type: getStringPtr("TEXT")},
                {Name: getStringPtr("NEW_SECRET"), Value: getStringPtr("some secret"), Type: getStringPtr("SECURE")},
            },
        },
    }

    for _, c := range testcases {
        actual := makeEnvPatch(c.currentEnv, c.textEnv, c.secretEnv, c.deletedKeys)

        sort.Slice(actual, func(i, j int) bool {
            return *actual[i].Name < *actual[j].Name
        })

        assert.Equal(t, actual, c.expected)
    }
}
