# srlinux-sync
This repository contains the needed components to synchronizes state paths between two or more SR Linux devices. 

This can be used to synchronize certain state paths between an instance of SR Linux running in production, and a representative instance inside the Fabric Services System Digital Sandbox, or any other simulation tool for the purposes of CI, as one example use case.

It contains an NDK agent that exposes a configuration path that can be set over gNMI (the trigger to the simulated node that a path has changed value), and two event handler scripts. The first event handler script - the 'sync' script listens to this path and takes an action accordingly. Event handler is used here for its interactions with the ephemeral data store - allowing state paths to be overriden by 'configuration'. The second script - the 'trigger' acts as the gNMI setter - setting the exposed NDK path on the simulated node as a result of a path changing value on the production node.

```
       ┌───────────┐     ┌───────────┐
       │           │     │           │
       │   srl-1   │     │   srl-2   │
       │           │     │           │
       └─────┬─────┘     └─────┬─────┘
             │eth1             │eth1
             │                 │
             │                 │
```

Taking the above topology, and assuming there is a desire to synchronize the interface state between srl-1 eth1 and srl-2 eth1 - where srl-1 is the production instance, and srl-2 is the digital twin, the agent and event handler configs would function as per the below.

* srlinux-sync NDK agent, and event handler 'sync' script installed on srl-2 - this receives the trigger from srl-1 and reacts accordingly
* Event handler 'trigger' script installed on srl-1 - this generates the gNMI set request when a path changes value
* 'trigger' script provided the below configuration:
# Example config:
```
system {
    event-handler {
        instance ds-trigger {
            admin-state enable
            upython-script ds-trigger.py
            paths [
                "interface ethernet-1/{1,2} oper-state"
            ]
            options {
                object target {
                    value 192.168.0.1:57400
                }
                object login {
                    value admin
                }
                object password {
                    value NokiaSrl1!
                }
            }
        }
    }
}
```

# Building
The NDK agent can be built from source using the provided `Makefile`. Options available are:

* `make help`
Provides help around how to use the `Makefile`

* `make fmt`
Runs `gofmt` on all Go source files

* `make lint`
Runs `golint` on all Go source files

* `make build`
This builds the NDK agent using a local Go installation

* `make rpm`
This builds the agent, then packages it and other required artifacts into an RPM using the `goreleaser/nfpm` container

* `make image`
This uses a docker container to build the agent, and packages binaries and other artifacts inside the container with an entrypoint that copies these to the host

* `make clean`
This cleans the workspace, removing artifacts from previous builds

# Installation
## via rpm
All steps are executed as the admin user, starting from a bash shell.

Install the RPM on both the the Digital Sandbox and production target nodes
`sudo rpm -ivh dssync-1.0.0.x86_64.rpm`
### On the Digital Sandbox node
Reload app_mgr to pick up the new dssync agent YANG
`sr_cli tools system app-management application app_mgr reload`
Enable the DS synchronization event handler
`sr_cli -ec system event-handler instance ds-sync admin-state enable upython-script ds-sync.py paths [ \"dssync override \* value\" ]`
This results in the following configuration:
```
system {
    event-handler {
        instance ds-sync {
            admin-state enable
            upython-script ds-sync.py
            paths [
                "dssync override * value"
            ]
        }
    }
}
```

### On the production node
Enable the DS trigger event handler
`sr_cli -ec system event-handler instance ds-trigger admin-state enable upython-script ds-trigger.py paths [ \"interface ethernet-1/\{1,2\} oper-state\" ] options object target value 172.20.20.2:57400`
This results in the following configuration:
```
system {
    event-handler {
        instance ds-trigger {
            admin-state enable
            upython-script ds-trigger.py
            paths [
                "interface ethernet-1/{1,2} oper-state"
            ]
            options {
                object target {
                    value 172.20.20.2:57400
                }
            }
        }
    }
}
```

If required replace the `paths` list with the set of interfaces you wish to synchronize state between production and the Digital Sandbox targets, and set the `target` value to the correct value of the Digital Sandbox target. This address is the address and port the Digital Sandbox target runs a gNMI server instance on.
