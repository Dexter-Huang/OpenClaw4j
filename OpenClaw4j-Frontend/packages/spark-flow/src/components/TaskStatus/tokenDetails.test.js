/* eslint-env jest */

const { getTaskTokenDetails } = require('./tokenDetails');

describe('getTaskTokenDetails', () => {
  it('ignores workflow nodes whose usages field is null', () => {
    const details = getTaskTokenDetails([
      {
        node_id: 'Start_nPvj',
        node_name: '开始',
        node_type: 'Start',
        usages: null,
      },
      {
        node_id: 'LLM_5lKy',
        node_name: '大模型',
        node_type: 'LLM',
        usages: [
          {
            prompt_tokens: 3,
            completion_tokens: 4,
            total_tokens: 7,
          },
        ],
      },
    ]);

    expect(details).toEqual([
      {
        id: 'LLM_5lKy',
        name: '大模型',
        type: 'LLM',
        input: 3,
        output: 4,
      },
    ]);
  });
});
