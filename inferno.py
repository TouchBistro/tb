#!/usr/bin/env python
import yaml
import pprint
import sys
import os
import boto3

config = {}
config_out = {}
container_defs = []

ecs = boto3.client("ecs")

task_repo = "651264383976.dkr.ecr.us-east-1.amazonaws.com/ephemeral"

def listify(d):
	listed = []
	for key in d:
		listed.append({"name": key, "value":str(d[key])})
	return listed

def main():
	with open(sys.argv[1]) as stream:
		try:
			config = yaml.load(stream)
		except yaml.YAMLError as exc:
			print(exc)
	resolved = False
	servicelist = {}
	for serv in sys.argv[2:]:
		servicelist[serv] = True
	while not resolved:
		resolved = True
		thislist = dict(servicelist)
		for key in thislist:
			config_out[key] = config["services"][key]
			if "depends_on" in config["services"][key]:
				for dep in config["services"][key]["depends_on"]:
					name = os.path.expandvars(dep)
					if name not in servicelist:
						servicelist[name] = True
						resolved = False

	for name in config_out:
		service = config_out[name]
		new_container = {}
		migrations = False
		new_container["name"] = name
		new_container["image"] = os.path.expandvars(service["image"]) #resolve env here?
		if "command" in service:
			new_container["command"] = []
		new_container["dependsOn"] = []
		new_container["links"] = []
		if "depends_on" in service:
			for dep in service["depends_on"]:
				resolveddep = os.path.expandvars(dep)
				new_container["dependsOn"].append({"containerName": resolveddep, "condition": "START"})
				new_container["links"].append(resolveddep + ":" + resolveddep)
		new_container["environment"] = {}
		if "env_file" in service:
			for filename in service["env_file"]:
				with open(os.path.expandvars(filename)) as f:
					for line in f:
						if line.startswith('#') or line == '\n':
							continue
						key, value = line.strip().split('=', 1)
						new_container["environment"][key] = value
		if "environment" in service:
			for key in service["environment"]:
				new_container["environment"][key] = os.path.expandvars(str(service["environment"][key]))
				if key == "DB_HOST":
					migrations = True
			new_container["environment"]["START_SERVER"] = "true"
		new_container["environment"] = listify(new_container["environment"])
		new_container["portMappings"] = []
		#if "ports" in service and new_container["name"] != 'localstack':
		#	for portset in service["ports"]:
		#		new_container["portMappings"].append({"containerPort": int(portset.split(":")[1]), "hostPort": int(portset.split(":")[0])})
		migration_cmd = ""
		entrypoint_cmd = ""
		if migrations:
			migration_cmd = "yarn db:prepare && yarn serve"
		if "entrypoint" in service:
			entrypoint_cmd = " ".join(service["entrypoint"])
		if migration_cmd != "" or entrypoint_cmd != "":
			new_container["command"] = ["sh", "-c", str(entrypoint_cmd + " " + migration_cmd)]
		container_defs.append(new_container)

	timer_container = {
		"name": "ephemeral_timer",
		"image": "alpine",
		"command": ["sh", "-c", "sleep 600"]
	}
	container_defs.append(timer_container)

	for container in container_defs:
		print(container["name"] + ":")

	try:
		task = ecs.register_task_definition(family="ephemeral", \
			networkMode="bridge", containerDefinitions=container_defs,
			tags=[{"key": "ephemeral_env", "value": sys.argv[1]}],\
			requiresCompatibilities=["EC2"], memory="2048", cpu="1024")
		print(task)
	except ecs.exceptions.ClientException as e:
		print(e.message)

if __name__ == "__main__":
	main()