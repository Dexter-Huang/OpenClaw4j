const { getUsageSummary } = require('./usageSummary');

describe('getUsageSummary', () => {
  it('treats null usages as an empty usage list', () => {
    expect(getUsageSummary(null)).toEqual({
      input: 0,
      output: 0,
      total: 0,
    });
  });

  it('aggregates token usage values', () => {
    expect(
      getUsageSummary([
        {
          prompt_tokens: 2,
          completion_tokens: 3,
          total_tokens: 5,
        },
        {
          prompt_tokens: 7,
          completion_tokens: 11,
          total_tokens: 18,
        },
      ]),
    ).toEqual({
      input: 9,
      output: 14,
      total: 23,
    });
  });
});
