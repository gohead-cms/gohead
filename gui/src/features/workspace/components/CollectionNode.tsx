import { Handle, NodeProps, Position } from '@xyflow/react';
import { CollectionNodeType } from '../../../shared/types/collections';

// Use the CollectionNodeData interface with NodeProps
export function CollectionNode({ data }: NodeProps<CollectionNodeType>) {
  // Now, TypeScript knows `data.label` is a string, and `data.attributes` is an array of attributes
  const attributes = Array.isArray(data?.attributes) ? data.attributes : [];

  return (
    <div
      style={{
        background: '#fff',
        border: '2px solid #52b4ca',
        borderRadius: 8,
        padding: 16,
        minWidth: 180,
        boxShadow: '0 2px 10px #0001',
      }}
    >
      <div style={{ fontWeight: 700, marginBottom: 8 }}>{data.label}</div>
      <ul style={{ padding: 0, margin: 0, listStyle: 'none' }}>
        {attributes.map((attr) => (
          <li key={attr.name} style={{ fontSize: 13, marginBottom: 4 }}>
            <span style={{ fontWeight: 500 }}>{attr.name}</span>
            <span style={{ color: '#888', marginLeft: 8 }}>{attr.type}</span>
          </li>
        ))}
      </ul>
      <Handle type="source" position={Position.Right} />
      <Handle type="target" position={Position.Left} />
    </div>
  );
}