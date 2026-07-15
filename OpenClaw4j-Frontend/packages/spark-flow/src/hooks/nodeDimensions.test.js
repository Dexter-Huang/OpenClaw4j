/* eslint-env jest */

const { applyDimensionChanges } = require('./nodeDimensions');

describe('applyDimensionChanges', () => {
  it('keeps the original nodes reference when measured dimensions are unchanged', () => {
    const nodes = [
      {
        id: 'node-1',
        measured: {
          width: 120,
          height: 80,
        },
      },
    ];

    const result = applyDimensionChanges(nodes, [
      {
        id: 'node-1',
        type: 'dimensions',
        dimensions: {
          width: 120,
          height: 80,
        },
      },
    ]);

    expect(result).toBe(nodes);
    expect(result[0]).toBe(nodes[0]);
  });

  it('updates only nodes whose measured dimensions changed', () => {
    const nodes = [
      {
        id: 'node-1',
        measured: {
          width: 120,
          height: 80,
        },
      },
      {
        id: 'node-2',
        measured: {
          width: 90,
          height: 60,
        },
      },
    ];

    const result = applyDimensionChanges(nodes, [
      {
        id: 'node-1',
        type: 'dimensions',
        dimensions: {
          width: 121,
          height: 80,
        },
      },
      {
        id: 'node-2',
        type: 'dimensions',
        dimensions: {
          width: 90,
          height: 60,
        },
      },
    ]);

    expect(result).not.toBe(nodes);
    expect(result[0]).toEqual({
      id: 'node-1',
      measured: {
        width: 121,
        height: 80,
      },
    });
    expect(result[1]).toBe(nodes[1]);
  });
});
