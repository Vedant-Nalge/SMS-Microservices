package com.sms.sender.service;

import com.sms.sender.model.Models;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;

/**
 * Mocks the third-party SMS vendor API (e.g. imiconnect).
 * In production this would call the real vendor REST endpoint.
 */
@Slf4j
@Service
public class SmsVendorService {

    @Value("${app.sms.mock-success-rate:0.9}")
    private double mockSuccessRate;

    public Models.VendorResult send(String phoneNumber, String message) {
        log.info("Calling mock SMS vendor for phoneNumber={}", phoneNumber);

        // Simulate network latency
        simulateLatency();

        boolean success = ThreadLocalRandom.current().nextDouble() < mockSuccessRate;

        if (success) {
            String vendorMsgId = "VND-" + UUID.randomUUID().toString().substring(0, 8).toUpperCase();
            log.info("Mock vendor: SMS sent successfully, vendorMessageId={}", vendorMsgId);
            return Models.VendorResult.builder()
                    .success(true)
                    .status("SUCCESS")
                    .vendorMessageId(vendorMsgId)
                    .build();
        } else {
            log.warn("Mock vendor: SMS delivery failed for phoneNumber={}", phoneNumber);
            return Models.VendorResult.builder()
                    .success(false)
                    .status("FAILED")
                    .errorDetails("Mock vendor: delivery failure (simulated)")
                    .build();
        }
    }

    private void simulateLatency() {
        try {
            long millis = ThreadLocalRandom.current().nextLong(50, 200);
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}
