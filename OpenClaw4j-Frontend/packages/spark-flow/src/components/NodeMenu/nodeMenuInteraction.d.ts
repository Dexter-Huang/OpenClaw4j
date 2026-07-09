export interface INodeMenuItemInteractionParams {
  disableDrag?: boolean;
  nodesReadOnly: boolean;
}

export interface INodeMenuItemInteraction {
  draggable: boolean;
  disabled: boolean;
  shouldWarnReadonly: boolean;
}

export function getNodeMenuItemInteraction(
  params: INodeMenuItemInteractionParams,
): INodeMenuItemInteraction;
