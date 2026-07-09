function getNodeMenuItemInteraction({ disableDrag, nodesReadOnly }) {
  return {
    draggable: !disableDrag && !nodesReadOnly,
    disabled: nodesReadOnly,
    shouldWarnReadonly: nodesReadOnly,
  };
}

module.exports = {
  getNodeMenuItemInteraction,
};
