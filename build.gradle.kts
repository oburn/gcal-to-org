import com.github.benmanes.gradle.versions.updates.DependencyUpdatesTask

buildscript {
    repositories {
        mavenCentral()
    }
}

plugins {
    id("org.jetbrains.kotlin.jvm") version "2.2.0"
    id("com.github.ben-manes.versions") version "0.52.0"
    id("com.gradleup.shadow") version "8.3.8"
    //id("io.gitlab.arturbosch.detekt") version "1.21.0"
    //id("com.diffplug.spotless") version "6.10.0"
}

repositories {
    mavenCentral()
}

version = "1.0.0-SNAPSHOT"

dependencies {
    implementation("org.jetbrains.kotlin:kotlin-stdlib")
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation("com.google.apis:google-api-services-gmail:v1-rev20250630-2.0.0")
    implementation("com.google.http-client:google-http-client-gson:1.47.1")
    implementation("com.google.oauth-client:google-oauth-client-jetty:1.39.0")
    implementation("javax.mail:javax.mail-api:1.6.2")
    implementation("com.sun.mail:javax.mail:1.6.2")
    implementation("info.picocli:picocli:4.7.7")
    implementation("io.github.oshai:kotlin-logging-jvm:7.0.10")
    implementation("ch.qos.logback:logback-classic:1.5.18")
    implementation("ch.qos.logback:logback-core:1.5.18")
    implementation("io.github.resilience4j:resilience4j-retry:2.3.0")
    testImplementation("com.nhaarman:mockito-kotlin:1.6.0")
    testImplementation("io.mockk:mockk:1.14.5")
    testImplementation("org.junit.jupiter:junit-jupiter:5.13.4")
    testImplementation("org.junit.platform:junit-platform-launcher:1.13.4")
}

tasks.test {
    useJUnitPlatform()
}

fun isNonStable(version: String): Boolean {
    val stableKeyword = listOf("RELEASE", "FINAL", "GA").any { version.uppercase().contains(it) }
    val regex = "^[0-9,.v-]+(-r)?$".toRegex()
    val isStable = stableKeyword || regex.matches(version)
    return isStable.not()
}

tasks.withType<DependencyUpdatesTask> {
    rejectVersionIf {
        isNonStable(candidate.version)
    }
}
tasks.withType<org.gradle.jvm.tasks.Jar> {
    manifest {
        attributes["Main-Class"] = "gcal2org.RunnerKt"
    }
}

//detekt {
//    config = files("config/detekt/detekt.yml")
//    buildUponDefaultConfig = true
//}
//
//spotless {
//    kotlin {
//        ktlint()
//            .editorConfigOverride(
//                mapOf(
//                    "ij_kotlin_imports_layout" to "*",
//                    "disabled_rules" to "filename",
//                    "max_line_length" to 100
//                )
//            )
//    }
//}
//
