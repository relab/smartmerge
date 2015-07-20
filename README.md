# smartMerge

This should be adapted later to use gorums or gorums-grpc

Possible optimizations not yet implemented:
- QuorumRPCs could return ids of the responding processes. This way we could avoid contacting processes twice as part of different configurations.

- Instead of sending complete blueprints over the network, we could try to only send changes relative to the current blueprint. This is possible, since we already include the current blueprint in replies, if it differs from the clients current blueprint.

TODO: Implement reliable broadcast, by calling SetCurAsync on finding a new cur.

