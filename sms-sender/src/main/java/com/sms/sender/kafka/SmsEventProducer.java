package com.sms.sender.kafka;

import com.sms.sender.model.Models;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.support.SendResult;
import org.springframework.stereotype.Service;

import java.util.concurrent.CompletableFuture;

@Slf4j
@Service
@RequiredArgsConstructor
public class SmsEventProducer {

    private final KafkaTemplate<String, Models.SmsEvent> kafkaTemplate;

    @Value("${app.kafka.topic.sms-events}")
    private String smsEventsTopic;

    /**
     * Publishes an SMS event to Kafka. Uses messageId as the partition key so that
     * all events for the same message go to the same partition (ordering guarantee).
     */
    public void publishSmsEvent(Models.SmsEvent event) {
        log.info("Publishing SMS event to Kafka: messageId={}, userId={}, status={}",
                event.getMessageId(), event.getUserId(), event.getStatus());

        CompletableFuture<SendResult<String, Models.SmsEvent>> future =
                kafkaTemplate.send(smsEventsTopic, event.getMessageId(), event);

        future.whenComplete((result, ex) -> {
            if (ex != null) {
                log.error("Failed to publish SMS event to Kafka: messageId={}, error={}",
                        event.getMessageId(), ex.getMessage(), ex);
            } else {
                log.info("SMS event published successfully: messageId={}, partition={}, offset={}",
                        event.getMessageId(),
                        result.getRecordMetadata().partition(),
                        result.getRecordMetadata().offset());
            }
        });
    }
}
