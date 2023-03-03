import os
import graphviz
import subprocess
import json
import time

describe_tag = subprocess.check_output(["git", "describe", "--tags"]).decode("utf-8").strip()
title = "msh package graph (" + describe_tag + ")"

graph = graphviz.Digraph(comment=title)

graph.graph_attr["label"] = title
graph.graph_attr["labelloc"] = "t"	# title on top

visitedRoot = {}

def main():
	timestart = time.time()

	getImports("")
	print(json.dumps(visitedRoot, indent=2, sort_keys=True))

	graph.render(os.path.join(os.path.realpath(os.path.dirname(__file__)), "package-graph"), format="png", view=True, cleanup=True)

	print("execution time: %.3f seconds" % float(time.time()-timestart))

def getImports(rootAddr: str):
	# rootAddr:	msh/lib/errco
	# rootName:	errco
	if rootAddr == "":
		rootName = "main"
	else:
		rootName = rootAddr.replace("msh/lib/", "")
	
	if rootAddr in visitedRoot:
		# return if rootAddr is already visited
		return
	else:
		# initialize list of visited packages of rootAddr
		visitedRoot[rootAddr] = []
	
	graph.node(rootAddr, label=rootName)

	print("analyzing %s" % rootAddr)

	packages_string = subprocess.check_output(["go", "list", "-f", "{{ .Imports }}", rootAddr]).decode("utf-8")
	packages = packages_string.replace("[", "").replace("]", "").split()
	packagesAddr = [e for e in packages if "msh" in e]

	for packageAddr in packagesAddr:
		packageName = packageAddr.replace("msh/lib/", "")
		graph.node(packageAddr, label=packageName)
		graph.edge(rootAddr, packageAddr)
		
		getImports(packageAddr)

		visitedRoot[rootAddr].append(packageAddr)
	
	# print(visitedRoot)

if __name__ == "__main__":
	main()