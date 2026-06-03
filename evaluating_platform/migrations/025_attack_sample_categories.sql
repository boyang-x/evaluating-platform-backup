DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'attack_samples'
    ) THEN
        ALTER TABLE attack_samples DROP CONSTRAINT IF EXISTS attack_samples_sub_type_check;
        ALTER TABLE attack_samples ADD CONSTRAINT attack_samples_sub_type_check
            CHECK (sub_type IN (
                'prompt_injection',
                'jailbreak_question',
                'harmful_content',
                'privacy_sensitive',
                'fraud_social_engineering',
                'cyber_abuse',
                'bias_discrimination',
                'tool_abuse',
                'compliance_boundary',
                'direct_injection',
                'malicious_instruction',
                'compliance_detection',
                'malicious_poisoning'
            ));
    END IF;
END $$;
