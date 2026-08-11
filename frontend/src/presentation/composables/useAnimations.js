<script setup>
import { onMounted } from 'vue'
import gsap from 'gsap'
import { ScrollTrigger } from 'gsap/ScrollTrigger'

gsap.registerPlugin(ScrollTrigger)

const initAnimations = () => {
  // Hero Animation
  gsap.from('.hero-content > *', {
    y: 30,
    opacity: 0,
    duration: 0.8,
    stagger: 0.1,
    ease: 'power3.out'
  })

  // Floating illustration
  gsap.to('.hero-illustration img', {
    y: -12,
    duration: 3,
    repeat: -1,
    yoyo: true,
    ease: 'sine.inOut'
  })

  // Sections fade in on scroll
  gsap.utils.toArray('section').forEach(section => {
    gsap.from(section, {
      scrollTrigger: {
        trigger: section,
        start: 'top 80%',
        toggleActions: 'play none none reverse'
      },
      y: 50,
      opacity: 0,
      duration: 0.8,
      ease: 'power3.out'
    })
  })

  // Stats counter animation
  gsap.utils.toArray('.stat-num').forEach(stat => {
    gsap.to(stat, {
      scrollTrigger: {
        trigger: stat,
        start: 'top 85%'
      },
      innerText: stat.getAttribute('data-target'),
      duration: 2,
      snap: { innerText: 1 },
      ease: 'power2.out'
    })
  })
}

export { initAnimations }