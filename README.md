# xlReg_go

The reg library for xlattice_go.

[xlReg](xlReg.html) is a tool, primarily intended for use in testing,
which facilitates the formation of clusters, groups of cooperating nodes.
On registration, a
client/member is issued a globally unique NodeID, a 256-bit random value.
Once it has an ID, the member can create and/or join clusters.  The cluster has
a maximum size set when it is created.  When members join the cluster they
register their two RSA public keys and either one or two IP addresses.
If the cluster only supports communications between members, members
register only one IP address.  If non-members, clients, are allowed to
communicate with the cluster, members register a second address for
that purpose.  When a member has completed registration, it can retrieve
the configuration data other members have registered.

The xlReg server, its clients, and the cluster members, are all
XLattice [nodes](https://jddixon.github.io/xlattice_go/node.html).


## On-line Documentation

More information on the **xlReg_go** project can be found [here](https://jddixon.github.io/xlReg_go)
