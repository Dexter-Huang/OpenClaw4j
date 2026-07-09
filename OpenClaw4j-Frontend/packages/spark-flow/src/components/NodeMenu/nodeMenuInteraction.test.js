/* eslint-env jest */

const { getNodeMenuItemInteraction } = require('./nodeMenuInteraction');

describe('getNodeMenuItemInteraction', () => {
  it('disables dragging and requests a readonly warning when the flow is readonly', () => {
    expect(
      getNodeMenuItemInteraction({
        disableDrag: false,
        nodesReadOnly: true,
      }),
    ).toEqual({
      draggable: false,
      disabled: true,
      shouldWarnReadonly: true,
    });
  });

  it('keeps normal drag behavior in editable drafts', () => {
    expect(
      getNodeMenuItemInteraction({
        disableDrag: false,
        nodesReadOnly: false,
      }),
    ).toEqual({
      draggable: true,
      disabled: false,
      shouldWarnReadonly: false,
    });
  });
});
