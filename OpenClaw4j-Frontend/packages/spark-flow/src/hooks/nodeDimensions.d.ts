import type { Node, NodeChange } from '@xyflow/react';

export function applyDimensionChanges<T extends Node>(
  nodes: T[],
  changes: NodeChange<T>[],
): T[];
