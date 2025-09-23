rootProject.name = "gcal-to-org"

/**
 * This allows us to use the `gradle.properties` file to set the versions of the plugins, notably the Kotlin plugin.
 */
pluginManagement {
    val kotlinVersion: String by settings
    val ktlintPluginVersion: String by settings

    plugins {
        id("org.jetbrains.kotlin.jvm") version kotlinVersion
        id("org.jlleitschuh.gradle.ktlint") version ktlintPluginVersion
    }
}
