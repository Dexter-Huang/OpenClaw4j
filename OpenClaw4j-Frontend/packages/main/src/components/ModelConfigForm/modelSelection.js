function findModelOptionByValue(modelOptions, value) {
  if (!value.model_id) return undefined;

  const options = modelOptions.flatMap((item) => item.options);
  const matchedByProvider = value.provider
    ? options.find(
        (option) =>
          option.extra.provider === value.provider &&
          option.extra.model_id === value.model_id,
      )
    : undefined;

  return (
    matchedByProvider ||
    options.find((option) => option.extra.model_id === value.model_id)
  );
}

module.exports = {
  findModelOptionByValue,
};
