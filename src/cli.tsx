import React, { useState, useEffect } from 'react';
import { render, Text, Box, useApp, useInput } from 'ink';
import TextInput from 'ink-text-input';
import { VertexAI, HarmCategory, HarmBlockThreshold } from '@google-cloud/vertexai';

// --- Vertex AI Configuration ---
// TODO: Replace with your actual project ID and location
const PROJECT_ID = process.env.VERTEXAI_PROJECT
const LOCATION = process.env.VERTEXAI_LOCATION
const MODEL_NAME = 'gemini-2.5-pro-preview-03-25';

// Initialize Vertex AI
const vertex_ai = new VertexAI({ project: PROJECT_ID, location: LOCATION });
const generativeModel = vertex_ai.getGenerativeModel({
    model: MODEL_NAME,
    // Optional safety settings (adjust as needed)
    safetySettings: [
        {
            category: HarmCategory.HARM_CATEGORY_HATE_SPEECH,
            threshold: HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
        },
        {
            category: HarmCategory.HARM_CATEGORY_DANGEROUS_CONTENT,
            threshold: HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
        },
        {
            category: HarmCategory.HARM_CATEGORY_SEXUALLY_EXPLICIT,
            threshold: HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
        },
        {
            category: HarmCategory.HARM_CATEGORY_HARASSMENT,
            threshold: HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
        },
    ],
    generationConfig: {
        maxOutputTokens: 8192,
        temperature: 1,
        topP: 0.95,
    },
});

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

    const sendMessage = async (text: string) => {
        if (!text.trim() || isLoading) return;

        const userMessage: Message = { role: 'user', text };
        setMessages(prev => [...prev, userMessage]);
        setInput('');
        setIsLoading(true);

        try {
            const chat = generativeModel.startChat({}); // Start a new chat session for each message or manage history
            const result = await chat.sendMessage(text);
            const candidates = result.response?.candidates; // Use optional chaining

            if (candidates && candidates.length > 0) {
                const candidate = candidates[0];
                // Explicitly check if candidate is not undefined before accessing its properties
                if (candidate) {
                    // Check for finishReason other than STOP
                    if (candidate.finishReason && candidate.finishReason !== 'STOP') {
                        const modelMessage: Message = { role: 'model', text: `[Blocked: ${candidate.finishReason}]` };
                        setMessages(prev => [...prev, modelMessage]);
                    // Use optional chaining for content and parts access
                    } else if (candidate.content?.parts?.[0]?.text) {
                        // We already checked candidate.content.parts[0].text exists, so direct access is safe here
                        const modelText = candidate.content.parts[0].text;
                        const modelMessage: Message = { role: 'model', text: modelText };
                        setMessages(prev => [...prev, modelMessage]);
                    } else {
                        const modelMessage: Message = { role: 'model', text: '[Received empty or non-text response]' };
                        setMessages(prev => [...prev, modelMessage]);
                    }
                } else {
                     // Handle the unlikely case where candidates[0] is undefined despite the length check
                     const modelMessage: Message = { role: 'model', text: '[Received undefined candidate]' };
                     setMessages(prev => [...prev, modelMessage]);
                 }
            } else {
                // Handle cases where response or candidates might be missing, or blocked content
                const blockedReason = result.response?.promptFeedback?.blockReason;
                const modelMessage: Message = { role: 'model', text: blockedReason ? `[Blocked: ${blockedReason}]` : '[No response or candidates received]' };
                setMessages(prev => [...prev, modelMessage]);
            }
        } catch (error) {
            console.error("Error sending message to Vertex AI:", error);
            const errorMessage: Message = { role: 'model', text: `[Error: ${error instanceof Error ? error.message : 'Unknown error'}]` };
            setMessages(prev => [...prev, errorMessage]);
        } finally {
            setIsLoading(false);
        }
    }; // End of sendMessage function

    return (
        <Box flexDirection="column" padding={1} borderStyle="round" borderColor="cyan">
            <Box flexDirection="column" flexGrow={1} marginBottom={1}>
                {/* Using index as key is generally discouraged, but acceptable for this simple example */}
                {/* For more complex scenarios, consider using unique IDs for messages */}
                {messages.map((msg, index) => (
                    // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
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
                    <Text>Enter message (Ctrl+C to exit):</Text>
                </Box>
                <TextInput
                    value={input}
                    onChange={setInput}
                    onSubmit={sendMessage}
                    placeholder="Type your message here..."
                />
            </Box>
        </Box>
    );
}; // End of ChatApp component

render(<ChatApp />);