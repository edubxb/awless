package rdf

import (
	"context"

	"github.com/google/badwolf/storage"
	"github.com/google/badwolf/triple"
	"github.com/google/badwolf/triple/node"
	"github.com/google/badwolf/triple/predicate"
)

func Compare(rootID string, local *Graph, remote *Graph) (*Graph, *Graph, error) {
	allextras := NewGraph()
	allmissings := NewGraph()
	allcommons := NewGraph()

	rootNode, err := node.NewNodeFromStrings("/region", rootID)
	if err != nil {
		return allextras, allmissings, err
	}

	maxCount := max(local.TriplesCount(), remote.TriplesCount())
	processing := make(chan *node.Node, maxCount)

	processing <- rootNode

	for len(processing) > 0 {
		select {
		case node := <-processing:
			extras, missings, commons, err := compareChildTriplesOf(node, local, remote)
			if err != nil {
				return allextras, allmissings, err
			}

			allextras.Add(extras...)
			allmissings.Add(missings...)
			allcommons.Add(commons...)

			for _, nextNodeToProcess := range commons {
				objectNode, err := nextNodeToProcess.Object().Node()
				if err != nil {
					return allextras, allmissings, err
				}
				processing <- objectNode
			}
		}
	}

	return allextras, allmissings, nil
}

func compareChildTriplesOf(root *node.Node, localGraph storage.Graph, remoteGraph storage.Graph) ([]*triple.Triple, []*triple.Triple, []*triple.Triple, error) {
	var extras, missings, commons []*triple.Triple

	locals, err := triplesForSubjectAndPredicate(localGraph, root, parentOf)
	if err != nil {
		return extras, missings, commons, err
	}

	remotes, err := triplesForSubjectAndPredicate(remoteGraph, root, parentOf)
	if err != nil {
		return extras, missings, commons, err
	}

	extras = append(extras, substractTriples(locals, remotes)...)
	missings = append(missings, substractTriples(remotes, locals)...)
	commons = append(commons, intersectTriples(locals, remotes)...)

	return extras, missings, commons, nil
}

func triplesForSubjectAndPredicate(graph storage.Graph, subject *node.Node, predicate *predicate.Predicate) ([]*triple.Triple, error) {
	errc := make(chan error)
	triplec := make(chan *triple.Triple)

	go func() {
		defer close(errc)
		errc <- graph.TriplesForSubjectAndPredicate(context.Background(), subject, predicate, storage.DefaultLookup, triplec)
	}()

	var triples []*triple.Triple

	for t := range triplec {
		triples = append(triples, t)
	}

	return triples, <-errc
}

func max(a, b int) int {
	if a < b {
		return b
	}

	return a
}
