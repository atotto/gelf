import { VertexAI, HarmCategory, HarmBlockThreshold } from '@google-cloud/vertexai';
import type { ChatSession } from '@google-cloud/vertexai';

const PROJECT_ID = process.env.VERTEXAI_PROJECT;
const LOCATION = process.env.VERTEXAI_LOCATION;
const MODEL_NAME = 'gemini-2.5-pro-preview-03-25';

if (!PROJECT_ID || !LOCATION) {
    throw new Error('VERTEXAI_PROJECT and VERTEXAI_LOCATION environment variables must be set.');
}

const vertex_ai = new VertexAI({ project: PROJECT_ID, location: LOCATION });
const generativeModel = vertex_ai.getGenerativeModel({
    model: MODEL_NAME,
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

let chat: ChatSession | null = null;

const getChatSession = (): ChatSession => {
    if (!chat) {
        chat = generativeModel.startChat({});
    }
    return chat;
};

/**
 * Sends a message to the Vertex AI model and returns the response.
 * @param text The user's message.
 * @returns The model's response text or an error/blocked message.
 */
export const sendMessageToVertexAI = async (text: string): Promise<string> => {
    const currentChat = getChatSession();
    try {
        const result = await currentChat.sendMessage(text);
        const candidates = result.response?.candidates;

        if (candidates && candidates.length > 0) {
            const candidate = candidates[0];
            if (candidate) {
                if (candidate.finishReason && candidate.finishReason !== 'STOP') {
                    return `[Blocked: ${candidate.finishReason}]`;
                }
                if (candidate.content?.parts?.[0]?.text) {
                    return candidate.content.parts[0].text;
                }
                return '[Received empty or non-text response]';
            }
            return '[Received undefined candidate]';
        }

        const blockedReason = result.response?.promptFeedback?.blockReason;
        return blockedReason ? `[Blocked: ${blockedReason}]` : '[No response or candidates received]';

    } catch (error) {
        console.error("Error sending message to Vertex AI:", error);
        return `[Error: ${error instanceof Error ? error.message : 'Unknown error'}]`;
    }
};