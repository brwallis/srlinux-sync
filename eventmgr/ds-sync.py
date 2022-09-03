#!/usr/bin/env python3
#
# Digital Sandbox synchronization
#
# This script provides the 'sync' portion of Digital Sandbox synchronization - it should run alongside the dssync agent in the Digital Sandbox instance of SR Linux, it should not run on the production node.
# It's logic is quite simple - it monitors the dssync agent's configuration, and when it is changed it takes the path provided as an override and the override value and does an ephemeral set towards the local SR Linux instance.
#
# The solution only supports interface oper-state synchronization currently.
#
# Paths:
#   dssync override value - e.g. dssync override "interface ethernet-1/1 oper-state" value
# Options:
#   No options are currently required or supported
#
# Example config:
#    system {
#        event-handler {
#            instance ds-sync {
#                admin-state enable
#                upython-script ds-sync.py
#                paths [
#                    "dssync override * value"
#                ]
#            }
#        }
#    }

import sys
import json

def extract_path(path):
    override_path = path[16:-6]
    return override_path

def event_handler_main(in_json_str):
    in_json = json.loads(in_json_str)
    paths = in_json['paths']

    response_actions = []
    for path in paths:
        override_path = extract_path(path.get('path'))
        response_actions.append({'set-ephemeral-path' : {'path':override_path,'value':path.get('value')}})

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
            "path": "dssync override interface ethernet-1/1 oper-state value",
            "value": "up"
        },
        {
            "path": "dssync override interface ethernet-1/1 oper-state value",
            "value": "down"
        }
    ]
}"""
    json_response = event_handler_main(example_in_json_str)
    print(f"Response JSON:\n{json_response}")


if __name__ == "__main__":
    sys.exit(main())