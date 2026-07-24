# Compose / Material3
-keep class androidx.compose.** { *; }
-dontwarn androidx.compose.**

# kotlinx-serialization（反射序列化需保留 @Serializable 类与 serializer）
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt
-keepclassmembers class kotlinx.serialization.json.** {
    *** Companion;
}
-keepclasseswithmembers class kotlinx.serialization.json.** {
    kotlinx.serialization.KSerializer serializer(...);
}
-keep,includedescriptorclasses class com.nekospeed.android.data.model.**$$serializer { *; }
-keepclassmembers class com.nekospeed.android.data.model.** {
    *** Companion;
}
-keepclasseswithmembers class com.nekospeed.android.data.model.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Coil
-dontwarn coil.**
-keep class coil.** { *; }

# Kotlin 协程
-dontwarn kotlinx.coroutines.**
