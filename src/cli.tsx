import React, { useState, useEffect } from 'react';
import { render, Text, Box, useApp, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { sendMessageToVertexAI } from './vertexai'; // Import the new function

interface Message {
    role: 'user' | 'model';
    text: string;
}

const ChatApp = () => {
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const { exit } = useApp();

    // Handle Ctrl+C to exit
    useInput((input, key) => {
        if (key.ctrl && input === 'c') {
            exit();
        }
    });

    const handleSendMessage = async (text: string) => {
        if (!text.trim() || isLoading) return;

        const userMessage: Message = { role: 'user', text };
        setMessages(prev => [...prev, userMessage]);
        setInput('');
        setIsLoading(true);

        try {
            const modelResponse = await sendMessageToVertexAI(text); // Use the imported function
            const modelMessage: Message = { role: 'model', text: modelResponse };
            setMessages(prev => [...prev, modelMessage]);
        } catch (error) {
            // Basic error handling for unexpected issues during the call itself
            console.error("Error in ChatApp calling sendMessageToVertexAI:", error);
            const errorMessage: Message = { role: 'model', text: `[App Error: ${error instanceof Error ? error.message : 'Unknown error'}]` };
            setMessages(prev => [...prev, errorMessage]);
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Box flexDirection="column" padding={1} borderStyle="round" borderColor="cyan">
            <Box flexDirection="column" flexGrow={1} marginBottom={1}>
                {/* Consider using unique IDs if messages can be deleted/reordered */}
                {messages.map((msg, index) => (
                    // biome-ignore lint/suspicious/noArrayIndexKey: Simple list, index is acceptable here
                    <Box key={index} marginBottom={1}>
                        <Text bold color={msg.role === 'user' ? 'blue' : 'green'}>
                            {msg.role === 'user' ? 'You: ' : 'AI: '}
                        </Text>
                        <Text>{msg.text}</Text>
                    </Box>
                ))}
                {isLoading && <Text color="yellow">AI is thinking...</Text>}
            </Box>
            <Box>
                <Box marginRight={1}>
                    <Text>Enter message:</Text>
                </Box>
                <TextInput
                    value={input}
                    onChange={setInput}
                    onSubmit={handleSendMessage} // Use the renamed handler
                    placeholder="Type your message here..."
                />
            </Box>
        </Box>
    );
};

render(<ChatApp />);