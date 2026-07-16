/* eslint-env jest */

const { findModelOptionByValue } = require('./modelSelection');

describe('model selection restore', () => {
  const modelOptions = [
    {
      label: 'DashScope',
      options: [
        {
          label: 'qwen-plus',
          value: 'dashscope@@@qwen-plus',
          extra: {
            provider: 'dashscope',
            model_id: 'qwen-plus',
            tags: [],
          },
        },
        {
          label: 'qwen-vl-plus',
          value: 'dashscope@@@qwen-vl-plus',
          extra: {
            provider: 'dashscope',
            model_id: 'qwen-vl-plus',
            tags: ['vision'],
          },
        },
      ],
    },
  ];

  it('restores the selected vision model from saved provider and model id', () => {
    const option = findModelOptionByValue(modelOptions, {
      provider: 'dashscope',
      model_id: 'qwen-vl-plus',
    });

    expect(option.extra.tags).toContain('vision');
  });

  it('falls back to model id for older saved data without provider', () => {
    const option = findModelOptionByValue(modelOptions, {
      model_id: 'qwen-vl-plus',
    });

    expect(option.value).toBe('dashscope@@@qwen-vl-plus');
  });
});
