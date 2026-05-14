# Transitions & Flow

Transitions define how the platform moves from one state to another.

## Automatic Transitions
By default, nodes transition to their `next` state once execution finishes successfully.

## Conditional Triggers
You can define custom triggers based on the output of a node:

- **Approved**: Transition to path A.
- **Rejected**: Transition to path B.
- **Retry**: Re-execute the current node with updated context.

## State Persistence (The Blackboard)
The **Blackboard** is a shared JSON object that follows the workflow from start to finish. Every transition can read from and write to the blackboard, ensuring that later steps have full context of what happened in previous nodes.
