/**
 * Dagre-based auto-layout for Vue Flow graphs.
 *
 * Usage:
 *   const { layout } = useLayout()
 *   nodes.value = layout(nodes.value, edges.value)
 */
import dagre from 'dagre'

const NODE_WIDTH = 180
const NODE_HEIGHT = 72

export function useLayout() {
  /**
   * Compute new node positions using dagre's Sugiyama layout.
   * Returns a new nodes array with updated `position` fields; edges unchanged.
   *
   * @param {object[]} nodes  Vue Flow node objects
   * @param {object[]} edges  Vue Flow edge objects
   * @param {'TB'|'LR'} direction  Top-to-bottom (default) or left-to-right
   */
  function layout(nodes, edges, direction = 'TB') {
    if (!nodes.length) return nodes

    const g = new dagre.graphlib.Graph()
    g.setDefaultEdgeLabel(() => ({}))
    g.setGraph({
      rankdir: direction,
      nodesep: 80,
      ranksep: 120,
      marginx: 60,
      marginy: 60,
    })

    nodes.forEach(n => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }))

    // Only add edges whose source and target both exist (prevents dagre crash on stale refs)
    const nodeIds = new Set(nodes.map(n => n.id))
    edges.forEach(e => {
      if (nodeIds.has(e.source) && nodeIds.has(e.target)) {
        g.setEdge(e.source, e.target)
      }
    })

    dagre.layout(g)

    return nodes.map(n => {
      const pos = g.node(n.id)
      return {
        ...n,
        position: {
          x: pos.x - NODE_WIDTH / 2,
          y: pos.y - NODE_HEIGHT / 2,
        },
      }
    })
  }

  return { layout }
}
