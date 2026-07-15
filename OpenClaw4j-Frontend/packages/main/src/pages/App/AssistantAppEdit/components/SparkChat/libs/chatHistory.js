function prepareMessagesForRegenerate(messages, fallbackUserMessage) {
  const nextMessages = [...messages];

  // 重新生成时 UI 会先移除答案卡片；这里同步裁剪请求历史里的旧答案，
  // 并复用被重新生成的 user 消息，避免发出 user -> assistant -> user 的重复上下文。
  while (
    nextMessages.length &&
    nextMessages[nextMessages.length - 1].role === 'assistant'
  ) {
    nextMessages.pop();
  }

  if (nextMessages[nextMessages.length - 1]?.role === 'user') {
    return nextMessages;
  }

  return [...nextMessages, fallbackUserMessage];
}

module.exports = {
  prepareMessagesForRegenerate,
};
