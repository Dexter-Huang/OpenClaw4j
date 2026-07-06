function getTaskTokenDetails(nodeResults) {
  const list = [];

  if (!Array.isArray(nodeResults)) {
    return list;
  }

  nodeResults.forEach((item) => {
    if (!Array.isArray(item?.usages)) {
      return;
    }

    const tokenMap = item.usages.reduce(
      (acc, cur) => {
        acc.input += cur?.prompt_tokens || 0;
        acc.output += cur?.completion_tokens || 0;
        return acc;
      },
      { input: 0, output: 0 },
    );

    list.push({
      id: item.node_id,
      name: item.node_name,
      type: item.node_type,
      ...tokenMap,
    });
  });

  return list;
}

module.exports = {
  getTaskTokenDetails,
};
