/* eslint-env jest */

const {
  MODEL_SELECTOR_DROPDOWN_WIDTH,
  MODEL_SELECTOR_WIDTH,
} = require('./modelSelectorLayout');

describe('model selector layout', () => {
  it('keeps the selected model visible in the API config header', () => {
    expect(MODEL_SELECTOR_WIDTH).toBeGreaterThanOrEqual(168);
  });

  it('uses a wider dropdown for model names', () => {
    expect(MODEL_SELECTOR_DROPDOWN_WIDTH).toBeGreaterThanOrEqual(240);
    expect(MODEL_SELECTOR_DROPDOWN_WIDTH).toBeGreaterThanOrEqual(
      MODEL_SELECTOR_WIDTH,
    );
  });
});
