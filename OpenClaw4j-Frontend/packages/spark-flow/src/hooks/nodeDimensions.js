function getDimensionChangeMap(changes) {
  return changes.reduce((map, change) => {
    if (change?.type === 'dimensions' && change?.dimensions) {
      map[change.id] = change.dimensions;
    }

    return map;
  }, {});
}

function isMeasuredSame(measured, dimensions) {
  return (
    measured?.width === dimensions.width && measured?.height === dimensions.height
  );
}

function applyDimensionChanges(nodes, changes) {
  const dimensionChangeMap = getDimensionChangeMap(changes);
  let hasChanged = false;

  const nextNodes = nodes.map((node) => {
    const dimensions = dimensionChangeMap[node.id];
    if (!dimensions || isMeasuredSame(node.measured, dimensions)) {
      return node;
    }

    hasChanged = true;
    return {
      ...node,
      measured: {
        width: dimensions.width,
        height: dimensions.height,
      },
    };
  });

  return hasChanged ? nextNodes : nodes;
}

module.exports = {
  applyDimensionChanges,
};
