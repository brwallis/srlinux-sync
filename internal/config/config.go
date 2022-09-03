package config

// import "fmt"

type Value struct {
	Value string `json:"value"`
}

type Address struct {
	Value string `json:"value"`
}

type Path struct {
	Value Value `json:"value"`
}

type YangOverride struct {
	Override Path `json:"override"`
}

// AgentYang holds the YANG schema for the agent
type AgentYang struct {
	Controller Address `json:"controller"`
}
