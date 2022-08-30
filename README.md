# srlinux-sync
This repository contains the needed components to synchronizes state paths between two or more SR Linux devices. 

This can be used to synchronize certain state paths between an instance of SR Linux running in production, and a representative instance inside the Fabric Services System Digital Sandbox, or any other simulation tool for the purposes of CI, as one example use case.

It contains an NDK agent that exposes a configuration path that can be set over gNMI (the trigger to the simulated node that a path has changed value), and two event handler scripts. The first event handler script - the 'sync' script listens to this path and takes an action accordingly. Event handler is used here for its interactions with the ephemeral data store - allowing state paths to be overriden by 'configuration'. The second script - the 'trigger' acts as the gNMI setter - setting the exposed NDK path on the simulated node as a result of a path changing value on the production node.

       ┌───────────┐     ┌───────────┐
       │           │     │           │
       │   srl-1   │     │   srl-2   │
       │           │     │           │
       └─────┬─────┘     └─────┬─────┘
             │eth1             │eth1
             │                 │
             │                 │

Taking the above topology, and assuming there is a desire to synchronize the interface state between srl-1 eth1 and srl-2 eth1 - where srl-1 is the production instance, and srl-2 is the digital twin, the agent and event handler configs would function as per the below.

* srlinux-sync NDK agent, and event handler 'sync' script installed on srl-2 - this receives the trigger from srl-1 and reacts accordingly
* Event handler 'trigger' script installed on srl-1 - this generates the gNMI set request when a path changes value
* 'trigger' script provided the below configuration:
# Example config:
#    system {
#        event-handler {
#            instance ds-trigger {
#                admin-state enable
#                upython-script ds-trigger.py
#                paths [
#                    "interface ethernet-1/{1,2} oper-state"
#                ]
#                options {
#                    object pair-address {
#                        value 192.168.0.1
#                    }
#                    object pair-login {
#                        value admin
#                    }
#                    object pair-password {
#                        value admin
#                    }
#                }
#            }
#        }
#    }
* Even


