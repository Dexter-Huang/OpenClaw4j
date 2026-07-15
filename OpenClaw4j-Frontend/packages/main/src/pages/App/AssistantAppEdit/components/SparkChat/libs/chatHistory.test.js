/* eslint-env jest */

const { prepareMessagesForRegenerate } = require('./chatHistory');

describe('chat history regeneration', () => {
  it('reuses the previous user message without sending the removed assistant reply', () => {
    const messages = [
      { role: 'user', content: '你好', content_type: 'text' },
      {
        role: 'assistant',
        content: '你好！请问有什么我可以帮你的吗？',
        content_type: 'text',
      },
    ];

    const nextMessages = prepareMessagesForRegenerate(messages, {
      role: 'user',
      content: '你好',
      content_type: 'text',
    });

    expect(nextMessages).toEqual([
      { role: 'user', content: '你好', content_type: 'text' },
    ]);
    expect(messages).toHaveLength(2);
  });

  it('keeps earlier turns and only removes the assistant answer being regenerated', () => {
    const messages = [
      { role: 'user', content: '第一问', content_type: 'text' },
      { role: 'assistant', content: '第一答', content_type: 'text' },
      { role: 'user', content: '第二问', content_type: 'text' },
      { role: 'assistant', content: '第二答', content_type: 'text' },
    ];

    expect(
      prepareMessagesForRegenerate(messages, {
        role: 'user',
        content: '第二问',
        content_type: 'text',
      }),
    ).toEqual([
      { role: 'user', content: '第一问', content_type: 'text' },
      { role: 'assistant', content: '第一答', content_type: 'text' },
      { role: 'user', content: '第二问', content_type: 'text' },
    ]);
  });
});
