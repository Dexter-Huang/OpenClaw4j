function getUsageSummary(usages) {
  if (!Array.isArray(usages)) {
    return { input: 0, output: 0, total: 0 };
  }

  return usages.reduce(
    (acc, usage) => {
      return {
        input: acc.input + (usage?.prompt_tokens || 0),
        output: acc.output + (usage?.completion_tokens || 0),
        total: acc.total + (usage?.total_tokens || 0),
      };
    },
    { input: 0, output: 0, total: 0 },
  );
}

module.exports = {
  getUsageSummary,
};
