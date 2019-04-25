package main

import (
		"io/ioutil";
		"log";
		"sync";
		"strings"
)

const ERR_MSG= "ERROR: %s"

/* metaRegistry
 * A key-value store dependency graph
 * useful for retrieving names of dependency packages
 *
 */
type MetaRegistry struct {
	dep_map map[string][]int // Mapping from handler names to dependency lists
	pckg_lut map[string][]int // A hashing table for IDs of package names
	dep_names []string
	handler_names []string
	cur_did int
	cur_hid int
	tickDID_lock sync.Mutex
	tickHID_lock sync.Mutex
}

/* Serializes the metaRegistry 
 * to the specified path
 *
 * argument - 
	path string - to directory path to serialize database to, or file path if ended in .mr
 */
func (mr *MetaRegistry) serialize(path string) int {
	 return 0
}

/* Opens a serialized metaRegistry
 * fills up the MetaRegistry that this is called from the serialized MR given as a path
 *
 * argument -
 *	path string - path to .mr file to open
 */
func (mr *MetaRegistry) open(path string) int {
	return 0
}

/* Initializes empty metaRegistry
 *
 */
func (mr *MetaRegistry) init() int {
	// Allocate hash table space
	mr.pckg_lut = make(map[string][]int)
	mr.dep_map = make(map[string][]int)
	return 0
}

/* Adds a new dependency relation into the metaRegistry
 * 
 */
func (mr *MetaRegistry) push(handler_name string, pckg_names []string) int {

	// First look at the handler, is it in the metaregistry?
	// If not then pack as following:
	// First index @0 - simply the handler id, reserved
	// Rest of the indicies - a list of dependency ids required by the handler
	// Dependency entries are to be packed in a similar manner, except First index refers to dep. id
	// and rest of indicies refer to handlers which refer to that dependency


	// First, Fill IDs if not assigned already
	mr.tickHID_lock.Lock()
	h := mr.dep_map[handler_name]

	if h == nil {
		log.Printf("INFO: Discovered new handler... create new handler node for %s with H-ID: %d", handler_name, mr.cur_hid)
		mr.dep_map[handler_name] = append(mr.dep_map[handler_name], mr.cur_hid)
		mr.handler_names = append(mr.handler_names, handler_name)
		mr.cur_hid++
	} else {
		log.Printf("ERROR: Handler %s already registered.", handler_name)
		mr.tickHID_lock.Unlock()
		return -1
	}
	mr.tickHID_lock.Unlock()

	for i := 0; i < len(pckg_names); i++ {
		mr.tickDID_lock.Lock()
		d := mr.pckg_lut[pckg_names[i]]
		if d == nil {
			log.Printf("INFO: Discovered new package... creating new dependency node for %s with D-ID: %d", pckg_names[i], mr.cur_did)
			mr.pckg_lut[pckg_names[i]] = append(mr.pckg_lut[pckg_names[i]], mr.cur_did)
			mr.dep_names = append(mr.dep_names, pckg_names[i])
			mr.cur_did++
		}
		mr.tickDID_lock.Unlock()
	}

	// Now IDs are filled. Scrape and add IDs to build the graph

	// Add dependencies to handler entry and add handler entry to dependencies
	for i := 0; i < len(pckg_names); i++ {
		mr.dep_map[handler_name] = append(mr.dep_map[handler_name], mr.pckg_lut[pckg_names[i]][0])
		mr.pckg_lut[pckg_names[i]] = append(mr.pckg_lut[pckg_names[i]], mr.dep_map[handler_name][0])
	}
	// success
	return 0
}

/* A wrapper to standard push
 * except it pushes based off a given cluster and handler name and performs file I/O
 * silently in the background
 * And uses the following assumptions:
 *	- cluster dir is in openLambda folder
 *	- all handlers reside in "registry"
 *	- anything the handler uses in in "packages.txt"
 */
func (mr *MetaRegistry) push_cluster_handler(handler_name string, clust_name string) int{
	clust_dir := "../" + clust_name + "/registry/" + handler_name + "/"
	istre, err := ioutil.ReadFile(clust_dir + "packages.txt")
	if err != nil {
		log.Fatal(err)
		return -1
	}

	full_in := string(istre)
	raw_package_list := strings.Split(full_in, "\n")

	// There is always one "EOF" tag along, ignore it
	if(len(raw_package_list) <= 1) {
		return -1
	}

	package_list := raw_package_list[0:len(raw_package_list) - 1]
	return mr.push(handler_name, package_list)
}

/* Given a handler_name
 * Returns a list of dependencies that this handler uses
 * If the item is not found, or an error occurs, returns nil
 */
func (mr *MetaRegistry) peek_handler_deps(handler_name string) []string{
	inds := mr.dep_map[handler_name]


	if inds == nil {
		return nil
	}

	var sl []string = nil

	for i := 1; i < len(inds); i++ {
		sl = append(sl, mr.dep_names[inds[i]])
	}

	return sl
}

func (mr *MetaRegistry) peek_deps_used_by(dep_name string) []string {
	inds := mr.pckg_lut[dep_name]

	if inds == nil {
		return nil
	}

	var hl []string = nil

	for i := 1; i < len(inds); i++ {
		hl = append(hl, mr.handler_names[inds[i]])
	}

	return hl
}
