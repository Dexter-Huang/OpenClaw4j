const React = require('react');

function UserTextCard(props) {
  return React.createElement(
    'div',
    {
      style: {
        whiteSpace: 'pre-wrap',
        tabSize: 4,
        overflowWrap: 'anywhere',
      },
    },
    props?.data?.content || '',
  );
}

function createUserTextCard(content) {
  return {
    code: 'Text',
    component: UserTextCard,
    data: {
      content: content || '',
    },
  };
}

function getUserMessageContent(message) {
  if (typeof message?.content === 'string' && message.content.length) {
    return message.content;
  }

  const textCard = message?.cards?.find((card) => card.code === 'Text');
  return textCard?.data?.content || '';
}

module.exports = {
  UserTextCard,
  createUserTextCard,
  getUserMessageContent,
};
