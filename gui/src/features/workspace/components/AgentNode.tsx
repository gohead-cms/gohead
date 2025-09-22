// src/components/AgentNode.tsx

import { Handle, NodeProps, Position } from '@xyflow/react';
import type { AgentNodeType } from '../../../shared/types';

// The component receives NodeProps with AgentNodeData as the data type
export default function AgentNode({ data }: NodeProps<AgentNodeType>) {
  const name = data.name || 'Unnamed Agent';

  const llmInfo = data.schema?.llmConfig
    ? `${data.schema.llmConfig.provider} (${data.schema.llmConfig.model})`
    : 'No LLM';

  const functions = Array.isArray(data.schema?.functions) ? data.schema.functions : [];

  const getTriggerDetails = () => {
    const trigger = data.schema?.trigger;
    if (!trigger) return 'No trigger configured';
    if (trigger.type === 'collection_event' && trigger.event_trigger) {
      const collection = trigger.event_trigger.collection;
      const events = trigger.event_trigger.events.join(', ');
      return `On '${collection}' [${events}]`;
    }
    return `Type: ${trigger.type}`;
  };

  return (
    <div
      style={{
        background: '#fff',
        border: '2px solid #8a52ca',
        borderRadius: 8,
        padding: 16,
        minWidth: 220,
        boxShadow: '0 2px 10px #0001',
        fontFamily: 'sans-serif',
      }}
    >
      <div style={{ fontWeight: 700, marginBottom: 12, fontSize: 16 }}>
        🤖 {name}
      </div>

      <div style={{ fontSize: 13, display: 'grid', gap: 8 }}>
        <div>
          <strong style={{ color: '#555' }}>Trigger:</strong>
          <span style={{ color: '#777', marginLeft: 8 }}>{getTriggerDetails()}</span>
        </div>
        <div>
          <strong style={{ color: '#555' }}>LLM:</strong>
          <span style={{ color: '#777', marginLeft: 8 }}>{llmInfo}</span>
        </div>
      </div>

      {functions.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <strong style={{ color: '#555', fontSize: 13 }}>Functions:</strong>
          <ul style={{ padding: '0 0 0 16px', margin: '4px 0 0 0' }}>
            {functions.map((func) => (
              <li key={func.name} style={{ fontSize: 12, color: '#777', marginBottom: 2 }}>
                {func.name}
              </li>
            ))}
          </ul>
        </div>
      )}

      <Handle type="source" position={Position.Right} />
      <Handle type="target" position={Position.Left} />
    </div>
  );
}