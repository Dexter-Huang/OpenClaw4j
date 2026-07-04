/* eslint-env jest */

const {
  UserTextCard,
  createUserTextCard,
  getUserMessageContent,
} = require('./userMessageCard');

describe('user message card', () => {
  it('keeps raw user text in a pre-wrapped Text card', () => {
    const content = 'def demo():\n\treturn 1';
    const card = createUserTextCard(content);

    expect(card.code).toBe('Text');
    expect(card.data.content).toBe(content);
    expect(card.component).toBe(UserTextCard);
  });

  it('renders raw line breaks and tab indentation without markdown parsing', () => {
    const content = '# title\n\tindented';
    const element = UserTextCard({ data: { content } });

    expect(element.type).toBe('div');
    expect(element.props.style.whiteSpace).toBe('pre-wrap');
    expect(element.props.style.tabSize).toBe(4);
    expect(element.props.children).toBe(content);
  });

  it('reads the original prompt from the raw Text card for regeneration', () => {
    const content = 'line 1\n    line 2';

    expect(
      getUserMessageContent({
        content: '',
        cards: [createUserTextCard(content)],
      }),
    ).toBe(content);
  });
});
