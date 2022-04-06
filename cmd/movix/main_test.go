package main

import "testing"

func TestMakeConfig(t *testing.T) {
    t.Run("Make a config and read it back", func(t *testing.T) {
        want := 4

        // Call the function you want to test.
        got := 2+2

        // Assert that you got your expected response
        if got != want {
            t.Fail()
        }
    })
}

func test_config() *Config {
	return &Config{
		Mediapath: "",
		Perm: 644,
		Dbpath: "",
		Treshhold: 0.9,
		LuaPluginPath: "",
	}
}

func TestWalk(t *testing.T) {
	// add file and see that it's in the 
}

func 
