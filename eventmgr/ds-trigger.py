#!/usr/bin/env python3
#
# Digital Sandbox trigger
#
# This script provides the 'trigger' portion of Digital Sandbox synchronization - it should run on the production node, that state needs to be synchronized from
# It's logic is quite simple - it monitors the provided paths, and sets their value as configuration to the dssync agent running in the Digital Sandbox instance of the node
#
# The solution only supports interface oper-state synchronization currently.
#
# Paths:
#   interface ethernet-1/{1,2} oper-state
# Options:
#   target: <value>[default = 172.20.20.2:57400] -  address:port to use as the Digital Sandbox target node
#   login: <value>[default = admin] - optional login to use when connecting to the Digital Sandbox target node
#   password: <value>[default = NokiaSrl1!] - optional password to use when connecting to Digital Sandbox target node
#
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
#                    object target {
#                        value 172.20.20.2:57400
#                    }
#                    object login {
#                        value admin
#                    }
#                    object password {
#                        value NokiaSrl1!
#                    }
#                    object network-instance {
#                        value mgmt
#                    }
#                }
#            }
#        }
#    }

import sys
import json

def get_target(options):
    return options.get('target', '172.20.20.2:57400')

def get_login(options):
    return options.get('login', 'admin')

def get_password(options):
    return options.get('password', 'NokiaSrl1!')

def get_network_instance(options):
    return options.get('network-instance', 'mgmt')

def extract_path(path):
    override_path = path[16:-6]
    return override_path

def event_handler_main(in_json_str):
    in_json = json.loads(in_json_str)
    paths = in_json['paths']
    options = in_json['options']

    target = get_target(options)
    login = get_login(options)
    password = get_password(options)
    network_instance = get_network_instance(options)

    response_actions = []
    for path in paths:
        response_actions.append({'run-script' : {"cmdline": f"sudo ip netns exec srbase-{network_instance} /opt/srlinux/dssync/bin/gnmic -a {target} -e json_ietf --skip-verify -u {login} -p {password} set --update-path /dssync/override[path={path.get('path')}]/value --update-value {path.get('value')}"}})

    response = {'actions':response_actions}
    return json.dumps(response)

#
# This code is only if you want to test it from bash - this isn't used when invoked from SRL
#
def main():
    example_in_json_str = """
{
    "paths": [
        {
            "path": "interface ethernet-1/1 oper-state",
            "value": "up"
        },
        {
            "path": "interface ethernet-1/2 oper-state",
            "value": "down"
        }
    ],
    "options": {
        "target": "172.20.20.2:57400"
    }
}"""
    json_response = event_handler_main(example_in_json_str)
    print(f"Response JSON:\n{json_response}")


if __name__ == "__main__":
    sys.exit(main())