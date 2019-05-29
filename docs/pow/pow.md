# Soteria's Proof-of-Work

*Read this document in other languages: [Chinese](pow_CN.md).*

This document is meant to outline, at a level suitable for someone without prior knowledge,
the algorithms and processes currently involved in Soteria's Proof-of-Work system. We'll start
with a general overview of cycles in a graph and the Cuckoo Cycle algorithm which forms the
basis of Soteria's proof-of-work. We'll then move on to Soteria-specific details, which will outline
the other systems that combine with Cuckoo Cycle to form the entirety of mining in Soteria. 

Please note that Soteria is currently under active development, and any and all of this is subject to
(and will) change before a general release. Also, Soteria PoW is heavily beased on Grin's PoW algorithm. 
For more information about Grin's PoW, please see [Grin's Documentation](https://github.com/mimblewimble/grin/blob/master/doc/pow/pow.md)

## Graphs and Cuckoo Cycle

Soteria's basic Proof-of-Work algorithm is called Cuckoo Cycle, which is specifically designed
to be resistant to Bitcoin style hardware arms-races. It is primarily a memory bound algorithm,
which, (at least in theory,) means that solution time is bound by memory bandwidth
rather than raw processor or GPU speed. As such, mining Cuckoo Cycle solutions should be viable on
most commodity hardware, and require far less energy than most other GPU, CPU or ASIC-bound
proof of work algorithms.

The Cuckoo Cycle POW is the work of John Tromp, and the most up-to-date documentation and implementations
can be found in [his github repository](https://github.com/tromp/cuckoo). The
[white paper](https://github.com/tromp/cuckoo/blob/master/doc/cuckoo.pdf) is the best source of
further technical details.

There is also a [podcast with Mike from Monero Monitor](https://moneromonitor.com/episodes/2017-09-26-Episode-014.html)
in which John Tromp talks at length about Cuckoo Cycle; recommended listening for anyone wanting
more background on Cuckoo Cycle, including more technical detail, the history of the algorithm's development
and some of the motivations behind it.

### Cycles in a Graph

Cuckoo Cycle is an algorithm meant to detect cycles in a bipartite graph of N nodes
and M edges. In plain terms, a bipartite graph is one in which edges (i.e. lines connecting nodes)
travel only between 2 separate groups of nodes. In the case of the Cuckoo hashtable in Cuckoo Cycle,
one side of the graph is an array numbered with odd indices (up to the size of the graph), and the other is numbered with even
indices. A node is simply a numbered 'space' on either side of the Cuckoo Table, and an Edge is a
line connecting two nodes on opposite sides. The simple graph below denotes just such a graph,
with 4 nodes on the 'even' side (top), 4 nodes on the odd side (bottom) and zero Edges
(i.e. no lines connecting any nodes.)

![alt text](images/cuckoo_base_numbered_minimal.png)

*A graph of 8 Nodes with Zero Edges*

Let's throw a few Edges into the graph now, randomly:

![alt text](images/cuckoo_base_numbered_few_edges.png)

*8 Nodes with 4 Edges, no solution*

We now have a randomly-generated graph with 8 nodes (N) and 4 edges (M), or an NxM graph where
N=8 and M=4. Our basic Proof-of-Work is now concerned with finding 'cycles' of a certain length
within this random graph, or, put simply, a series of connected nodes starting and ending on the
same node. So, if we were looking for a cycle of length 4 (a path connecting 4 nodes, starting
and ending on the same node), one cannot be detected in this graph.

Adjusting the number of Edges M relative to the number of Nodes N changes the difficulty of the
cycle-finding problem, and the probability that a cycle exists in the current graph. For instance,
if our POW problem were concerned with finding a cycle of length 4 in the graph, the current difficulty of 4/8 (M/N)
would mean that all 4 edges would need to be randomly generated in a perfect cycle (from 0-5-4-1-0)
in order for there to be a solution.

Let's add a few more edges, again at random:

![alt text](images/cuckoo_base_numbered_more_edges.png)

*8 Nodes with 7 Edges*

Where we can find a cycle:

![alt text](images/cuckoo_base_numbered_more_edges_cycle.png)

*Cycle Found from 0-5-4-1-0*

If you increase the number of edges relative to the number
of nodes, you increase the probability that a solution exists. With a few more edges added to the graph above,
a cycle of length 4 has appeared from 0-5-4-1-0, and the graph has a solution.

Thus, modifying the ratio M/N changes the number of expected occurrences of a cycle for a graph with
randomly generated edges.

For a small graph such as the one above, determining whether a cycle of a certain length exists
is trivial. But as the graphs get larger, detecting such cycles becomes more difficult. For instance,
does this graph have a cycle of length 8, i.e. 8 connected nodes starting and ending on the same node?

![alt text](images/cuckoo_base_numbered_many_edges.png)

*Meat-space Cycle Detection exercise*

The answer is left as an exercise to the reader, but the overall takeaways are:

* Detecting cycles in a graph becomes more difficult exercise as the size of a graph grows.
* The probability of a cycle of a given length in a graph increases as M/N becomes larger,
  i.e. you add more edges relative to the number of nodes in a graph.

### Cuckoo Cycle

The Cuckoo Cycle algorithm is a specialized algorithm designed to solve exactly this problem, and it does
so by inserting values into a structure called a 'Cuckoo Hashtable' according to a hash which maps nodes
into possible locations in two separate arrays. This document won't go into detail on the base algorithm, as
it's outlined plainly enough in section 5 of the
[white paper](https://github.com/tromp/cuckoo/blob/master/doc/cuckoo.pdf). There are also several
variants on the algorithm that make various speed/memory tradeoffs, again beyond the scope of this document.
However, there are a few details following from the above that we need to keep in mind before going on to more
technical aspects of Soteria's proof-of-work.

* The 'random' edges in the graph demonstrated above are not actually random but are generated by
  putting edge indices (0..N) through a seeded hash function, SIPHASH. Each edge index is put through the
  SIPHASH function twice to create two edge endpoints, with the first input value being 2 * edge_index,
  and the second 2 * edge_index+1. The seed for this function is based on a hash of a block header,
  outlined further below.
* The 'Proof' created by this algorithm is a set of nonces that generate a cycle of length 42,
  which can be trivially validated by other peers.
* Two main parameters, as explained above, are passed into the Cuckoo Cycle algorithm that affect the probability of a solution, and the
  time it takes to search the graph for a solution:
  * The M/N ratio outlined above, which controls the number of edges relative to the size of the graph.
    Cuckoo Cycle fixes M at N/2, which limits the number of cycles to a few at most.
  * The size of the graph itself

How these parameters interact in practice is looked at in more [detail below](#mining-loop-difficulty-control-and-timing).

Now, (hopefully) armed with a basic understanding of what the Cuckoo Cycle algorithm is intended to do, as well as the parameters that affect how difficult it is to find a solution, we move on to the other portions of our POW system.

